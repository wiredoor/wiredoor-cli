//go:build windows
// +build windows

package utils

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

const WIredoorServiceUserName = "WiredoorUser"

var (
	modNetapi32                 = windows.NewLazySystemDLL("netapi32.dll")
	procNetUserGetInfo          = modNetapi32.NewProc("NetUserGetInfo")
	procNetUserDel              = modNetapi32.NewProc("NetUserDel")
	procNetUserAdd              = modNetapi32.NewProc("NetUserAdd")
	procNetLocalGroupAddMembers = modNetapi32.NewProc("NetLocalGroupAddMembers")
	procNetApiBufferFree        = modNetapi32.NewProc("NetApiBufferFree")

	modAdvapi32            = windows.NewLazySystemDLL("advapi32.dll")
	procLookupAccountNameW = modAdvapi32.NewProc("LookupAccountNameW")

	modKernel32    = windows.NewLazySystemDLL("kernel32.dll")
	procLocalFree  = modKernel32.NewProc("LocalFree")
	procLocalAlloc = modKernel32.NewProc("LocalAlloc")
)

// Constantes y tamaños
const (
	ERROR_USER_NOT_FOUND = 2221
	LMEM_FIXED           = 0x0000
	LMEM_ZEROINIT        = 0x0040
)

// Estructuras
type USER_INFO_1 struct {
	usri1_name         *uint16
	usri1_password     *uint16
	usri1_password_age uint32
	usri1_priv         uint32
	usri1_home_dir     *uint16
	usri1_comment      *uint16
	usri1_flags        uint32
	usri1_script_path  *uint16
}

// USER_INFO_1_FLAGS
// (flags) usri1_flags (USER_INFO_1) o usri1008_flags (USER_INFO_1008)
// https://learn.microsoft.com/en-us/windows/win32/api/lmaccess/ns-lmaccess-user_info_1008
const (
	UF_SCRIPT                                 = 0x0001    // El script de inicio de sesión se ejecutará[citation:1]
	UF_ACCOUNTDISABLE                         = 0x0002    // La cuenta de usuario está deshabilitada[citation:1]
	UF_HOMEDIR_REQUIRED                       = 0x0008    // Se requiere directorio de inicio (se ignora)[citation:1]
	UF_PASSWD_NOTREQD                         = 0x0020    // No se requiere contraseña[citation:1]
	UF_PASSWD_CANT_CHANGE                     = 0x0040    // El usuario no puede cambiar la contraseña[citation:1]
	UF_LOCKOUT                                = 0x0010    // La cuenta está bloqueada[citation:1]
	UF_DONT_EXPIRE_PASSWD                     = 0x10000   // La contraseña nunca expira[citation:1]
	UF_ENCRYPTED_TEXT_PASSWORD_ALLOWED        = 0x0080    // Se permite contraseña con cifrado reversible[citation:1]
	UF_NOT_DELEGATED                          = 0x100000  // La cuenta no se puede delegar[citation:1]
	UF_SMARTCARD_REQUIRED                     = 0x40000   // Se requiere tarjeta inteligente[citation:1]
	UF_USE_DES_KEY_ONLY                       = 0x200000  // Usar solo cifrado DES para claves[citation:1]
	UF_DONT_REQUIRE_PREAUTH                   = 0x400000  // No requiere preautenticación Kerberos[citation:1]
	UF_TRUSTED_FOR_DELEGATION                 = 0x80000   // Habilitada para delegación[citation:1]
	UF_PASSWORD_EXPIRED                       = 0x800000  // La contraseña ha expirado[citation:1]
	UF_TRUSTED_TO_AUTHENTICATE_FOR_DELEGATION = 0x1000000 // Confiable para autenticar en delegación[citation:1]
	// Banderas de Tipo de Cuenta (elegir solo UNA)
	UF_NORMAL_ACCOUNT            = 0x0200 // Cuenta de usuario estándar[citation:1]
	UF_TEMP_DUPLICATE_ACCOUNT    = 0x0100 // Cuenta duplicada temporal[citation:1]
	UF_WORKSTATION_TRUST_ACCOUNT = 0x1000 // Cuenta de estación de trabajo[citation:1]
	UF_SERVER_TRUST_ACCOUNT      = 0x2000 // Cuenta de servidor[citation:1]
	UF_INTERDOMAIN_TRUST_ACCOUNT = 0x0800 // Cuenta de confianza entre dominios[citation:1]
)

type LOCALGROUP_MEMBERS_INFO_0 struct {
	lgrmi0_sid uintptr
}

// LocalAlloc wrapper con LMEM_ZEROINIT
func localAlloc(size uintptr) (uintptr, error) {
	r1, _, e1 := procLocalAlloc.Call(uintptr(LMEM_FIXED|LMEM_ZEROINIT), size)
	if r1 == 0 {
		if e1 != nil {
			return 0, e1
		}
		return 0, fmt.Errorf("LocalAlloc returned NULL")
	}
	return r1, nil
}

// LocalFree wrapper
func localFree(ptr uintptr) {
	if ptr == 0 {
		return
	}
	procLocalFree.Call(ptr)
}

// NetApiBufferFree wrapper
func netApiBufferFree(buf uintptr) {
	if buf == 0 {
		return
	}
	procNetApiBufferFree.Call(buf)
}

// lookupAccountName: obtiene PSID para un nombre de cuenta;
func lookupAccountName(name string) (uintptr, uint32, string, uint32, error) {
	// fmt.Printf("lookupAccountName for: %v\n", name)
	namePtr, err := windows.UTF16PtrFromString(name)
	if err != nil {
		return 0, 0, "", 0, fmt.Errorf("lookupAccountName UTF16PtrFromString: %v", err)
	}
	var sidSize uint32
	var domainLen uint32
	var useLocal uint32

	// primera llamada para tamaños
	r1, _, err := procLookupAccountNameW.Call(
		0,
		uintptr(unsafe.Pointer(namePtr)),
		0,
		uintptr(unsafe.Pointer(&sidSize)),
		0,
		uintptr(unsafe.Pointer(&domainLen)),
		uintptr(unsafe.Pointer(&useLocal)),
	)
	if r1 == 0 && err != nil {
		// last := windows.GetLastError()
		if err != windows.ERROR_INSUFFICIENT_BUFFER {
			return 0, 0, "", 0, fmt.Errorf("LookupAccountNameW (size probe) falló: %v", err)
		}
	}

	// si sidSize es 0, algo va mal
	if sidSize == 0 {
		return 0, 0, "", 0, fmt.Errorf("LookupAccountNameW devolvió sidSize=0")
	}

	// asignar PSID con LocalAlloc (LMEM_ZEROINIT)
	sidPtr, err := localAlloc(uintptr(sidSize))
	if err != nil {
		return 0, 0, "", 0, fmt.Errorf("LocalAlloc para SID falló: %v", err)
	}

	// domain buffer
	if domainLen == 0 {
		domainLen = 1
	}
	domainBuf := make([]uint16, domainLen+1)

	r1, _, e1 := procLookupAccountNameW.Call(
		0,
		uintptr(unsafe.Pointer(namePtr)),
		sidPtr,
		uintptr(unsafe.Pointer(&sidSize)),
		uintptr(unsafe.Pointer(&domainBuf[0])),
		uintptr(unsafe.Pointer(&domainLen)),
		uintptr(unsafe.Pointer(&useLocal)),
	)
	if r1 == 0 {
		// liberar memoria en caso de fallo
		localFree(sidPtr)
		last := windows.GetLastError()
		if last != syscall.Errno(0) {
			return 0, 0, "", 0, fmt.Errorf("LookupAccountNameW falló: %v", last)
		}
		if e1 != nil {
			return 0, 0, "", 0, fmt.Errorf("LookupAccountNameW falló: %v", e1)
		}
		return 0, 0, "", 0, fmt.Errorf("LookupAccountNameW falló sin GetLastError")
	}

	return sidPtr, sidSize, windows.UTF16ToString(domainBuf), useLocal, nil
}

// userExists, deleteUser, createUser (igual que antes, con x/sys/windows)
func userExists(username string) (bool, error) {
	u16, err := windows.UTF16PtrFromString(username)
	if err != nil {
		return false, err
	}
	var buf uintptr
	r1, _, _ := procNetUserGetInfo.Call(
		0,
		uintptr(unsafe.Pointer(u16)),
		0,
		uintptr(unsafe.Pointer(&buf)),
	)
	if r1 == 0 {
		netApiBufferFree(buf)
		return true, nil
	}
	if r1 == uintptr(ERROR_USER_NOT_FOUND) {
		return false, nil
	}
	return false, fmt.Errorf("NetUserGetInfo returned code %d", r1)
}

func DeleteUser(username string) error {
	u16, err := windows.UTF16PtrFromString(username)
	if err != nil {
		return err
	}
	r1, _, _ := procNetUserDel.Call(
		0,
		uintptr(unsafe.Pointer(u16)),
	)
	if r1 == 0 {
		return nil
	}
	if r1 == uintptr(ERROR_USER_NOT_FOUND) {
		return nil
	}
	return fmt.Errorf("NetUserDel returned code %d", r1)
}

func createUser(username, password string) error {
	namePtr, _ := windows.UTF16PtrFromString(username)
	passPtr, _ := windows.UTF16PtrFromString(password)
	user := USER_INFO_1{
		usri1_name:     namePtr,
		usri1_password: passPtr,
		usri1_priv:     1,
		usri1_flags:    UF_DONT_EXPIRE_PASSWD | UF_PASSWD_CANT_CHANGE | UF_NORMAL_ACCOUNT,
	}
	var parmErr uint32
	r1, _, _ := procNetUserAdd.Call(
		0,
		1,
		uintptr(unsafe.Pointer(&user)),
		uintptr(unsafe.Pointer(&parmErr)),
	)
	if r1 == 0 {
		return nil
	}
	return fmt.Errorf("NetUserAdd returned code %d, parmErr %d", r1, parmErr)
}

// addUserToLocalGroupBySID: usa LOCALGROUP_MEMBERS_INFO_0 (member por SID) y nombre localizado del grupo
func addUserToLocalGroupBySID(localizedGroupName string, memberSid uintptr) error {
	if memberSid == 0 {
		return fmt.Errorf("memberSid es nulo")
	}
	groupPtr, _ := windows.UTF16PtrFromString(localizedGroupName)
	member := LOCALGROUP_MEMBERS_INFO_0{
		lgrmi0_sid: memberSid,
	}
	r1, _, err := procNetLocalGroupAddMembers.Call(
		0,
		uintptr(unsafe.Pointer(groupPtr)),
		0, // level 0 (members by SID)
		uintptr(unsafe.Pointer(&member)),
		1,
	)
	if r1 == 0 {
		return nil
	}
	return fmt.Errorf("NetLocalGroupAddMembers returned code %d and error: %v", r1, err)
}

func generateSecurePasswordWindows(passwordLen int) string {
	if passwordLen < 8 {
		passwordLen = 12 // Mínimo recomendado para Windows
	}

	// Conjuntos de caracteres
	minusculas := "abcdefghijklmnopqrstuvwxyz"
	mayusculas := "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	numeros := "0123456789"
	simbolos := "!@#$%^&*()-_=+[]{}|;:,.<>?"

	// Combinar todos los caracteres
	todosCaracteres := minusculas + mayusculas + numeros + simbolos

	// Asegurar al menos un carácter de cada tipo
	password := make([]byte, passwordLen)

	// 5. Rellenar el resto con caracteres aleatorios
	for i := 0; i < passwordLen; i++ {
		idx, _ := rand.Int(rand.Reader, big.NewInt(int64(len(todosCaracteres))))
		password[i] = todosCaracteres[idx.Int64()]
	}

	// 6. Mezclar la contraseña para que no sea predecible
	for i := range password {
		j, _ := rand.Int(rand.Reader, big.NewInt(int64(len(password))))
		password[i], password[j.Int64()] = password[j.Int64()], password[i]
	}

	return string(password)
}

// -----------------------------

// Estructura necesaria para LsaOpenPolicy
type LSA_UNICODE_STRING struct {
	Length        uint16
	MaximumLength uint16
	Buffer        *uint16
}

type LSA_OBJECT_ATTRIBUTES struct {
	Length                   uint32
	RootDirectory            uintptr
	ObjectName               *LSA_UNICODE_STRING
	Attributes               uint32
	SecurityDescriptor       uintptr
	SecurityQualityOfService uintptr
}

var (
	advapi                  = syscall.NewLazyDLL("advapi32.dll")
	procLsaOpenPolicy       = advapi.NewProc("LsaOpenPolicy")
	procLsaAddAccountRights = advapi.NewProc("LsaAddAccountRights")
	procLsaClose            = advapi.NewProc("LsaClose")
)

// helper para crear LSA_UNICODE_STRING desde string
func newLSAString(s string) (*LSA_UNICODE_STRING, error) {
	u16, err := syscall.UTF16FromString(s)
	if err != nil {
		return nil, err
	}
	return &LSA_UNICODE_STRING{
		Length:        uint16((len(u16) - 1) * 2),
		MaximumLength: uint16(len(u16) * 2),
		Buffer:        &u16[0],
	}, nil
}

// GrantRightToSID otorga un derecho (privilegio) a un SID dado
func grantRightToSID(sid /*uintptr*/ *windows.SID, rightName string) error {
	var oa LSA_OBJECT_ATTRIBUTES
	oa.Length = uint32(unsafe.Sizeof(oa))
	var policyHandle uintptr

	// Abrir política
	ret, _, _ := procLsaOpenPolicy.Call(
		0, // NULL system name = local
		uintptr(unsafe.Pointer(&oa)),
		0x00000008|0x00000001|0x00000800, // POLICY_TRUST_ADMIN|POLICY_LOOKUP_NAMES | POLICY_CREATE_ACCOUNT
		uintptr(unsafe.Pointer(&policyHandle)),
	)
	if ret != 0 {
		// winErr, _, _ := procLsaNtStatusToWinError.Call(ret)
		// return fmt.Errorf("LsaOpenPolicy failed: NTSTATUS=0x%x, WinErr=%d", ret, winErr)
		return fmt.Errorf("LsaOpenPolicy failed: 0x%x", ret)
	}
	defer procLsaClose.Call(policyHandle)

	// Preparar el derecho
	lsaRight, err := newLSAString(rightName)
	if err != nil {
		return err
	}

	// LsaAddAccountRights
	ret, _, err = procLsaAddAccountRights.Call(
		policyHandle,
		uintptr(unsafe.Pointer(sid)),
		// sid,
		uintptr(unsafe.Pointer(lsaRight)),
		1, // count
	)
	if ret != uintptr(windows.STATUS_SUCCESS) {
		return fmt.Errorf("LsaAddAccountRights failed: 0x%x err:%v ", ret, err)
	}

	return nil
}

// -----------------------------
// Verifica y ajusta privilegios
func EnablePrivilege(privilege string) error {
	var token windows.Token
	currentProcess, _ := windows.GetCurrentProcess()
	defer windows.CloseHandle(currentProcess)

	err := windows.OpenProcessToken(currentProcess,
		windows.TOKEN_ADJUST_PRIVILEGES|windows.TOKEN_QUERY, &token)
	if err != nil {
		return err
	}
	defer token.Close()

	var tp windows.Tokenprivileges
	tp.PrivilegeCount = 1
	tp.Privileges[0].Attributes = windows.SE_PRIVILEGE_ENABLED

	// Obtener LUID del privilegio
	privilegeStr, _ := windows.UTF16PtrFromString(privilege)
	err = windows.LookupPrivilegeValue(nil, privilegeStr, &tp.Privileges[0].Luid)
	if err != nil {
		return err
	}

	// Ajustar privilegio
	return windows.AdjustTokenPrivileges(token, false, &tp, 0, nil, nil)
}

// Antes de llamar a GrantRightToSID:

//-----------------------------

func CreateServiceAccount(customUserName string) (string, string, error) {

	// err := EnablePrivilege("SeSecurityPrivilege")
	// log.Printf("err : %v\n", err)
	// err = EnablePrivilege("SeBackupPrivilege")
	// log.Printf("err : %v\n", err)
	// err = EnablePrivilege("SeRestorePrivilege")
	// log.Printf("err : %v\n", err)

	username := customUserName
	if len(username) <= 0 {
		username = "TestUserGo"
	}
	password := generateSecurePasswordWindows(17)
	// fmt.Printf("Passwd: %s\n", password)

	// 1) Remove old user
	exists, err := userExists(username)
	if err != nil {
		return "", "", fmt.Errorf("unable to determine if old user exists: %v", err)
	}
	if exists {
		if err := DeleteUser(username); err != nil {
			return "", "", fmt.Errorf("failed to remove old uder: %v", err)
		}
	}

	// 2) create new user
	if err := createUser(username, password); err != nil {
		return "", "", fmt.Errorf("unable to create new user: %v", err)
	}

	// 3) Get PSID (LookupAccountNameW) -> asigned using LocalAlloc; free using LocalFree
	userSidPtr, _, domain, _, err := lookupAccountName(username)

	if err != nil {
		return "", "", fmt.Errorf("error using LookupAccountNameW: %v", err)
	}

	if userSidPtr == 0 {
		return "", "", fmt.Errorf("null SID from LookupAccountNameW")
	}

	// clean memory
	defer localFree(userSidPtr)

	// 4) ADMIN Group SID

	adminSID, err := windows.CreateWellKnownSid(windows.WinBuiltinAdministratorsSid)

	if err != nil {
		return "", "", fmt.Errorf("error on step CreateWellKnownSid: %v", err)
	}

	// 5) get localized name of Admin group
	//group, domain,type,error
	groupName, _, _, err := adminSID.LookupAccount("") // empty string for local system

	if err != nil {
		return "", "", fmt.Errorf("error on step LookupAccountSidW: %v", err)
	}
	localized := groupName
	// do not use domain name or DOMAIN\Group format

	if localized == "" {
		return "", "", fmt.Errorf("invalid localized user name")
	}
	// 6) add SID to group using localized group name
	if err := addUserToLocalGroupBySID(localized, userSidPtr); err != nil {
		return "", "", fmt.Errorf("error añadiendo al grupo: %v", err)
	}

	// 7) add service priv
	sid, _, _, err := windows.LookupSID("", username)
	fmt.Printf("%s\n", sid.String())
	err = grantRightToSID(sid, "SeServiceLogonRight")
	if err != nil {
		return "", "", fmt.Errorf("error on step grantRightToSID: %v", err)
	}
	return domain + "\\" + username, password, nil
}
