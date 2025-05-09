package wiredoor

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/wiredoor/wiredoor-cli/version"
)

type apiRequest struct {
	Server  string
	Method  string
	Path    string
	Body    []byte
	Token   string
	Timeout int
}

type AdminCredentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type adminLoginResponse struct {
	Token     string `json:"token"`
	ExpiresIn string `json:"expiresIn"`
}

type HttpServiceParams struct {
	Name         string   `json:"name"`
	Domain       string   `json:"domain"`
	PathLocation string   `json:"pathLocation"`
	BackendHost  string   `json:"backendHost,omitempty"`
	BackendProto string   `json:"backendProto"`
	BackendPort  int      `json:"backendPort"`
	AllowedIps   []string `json:"allowedIps"`
	BlockedIps   []string `json:"blockedIps"`
}

type TcpServiceParams struct {
	Name        string   `json:"name"`
	Domain      string   `json:"domain,omitempty"`
	Proto       string   `json:"proto"`
	BackendHost string   `json:"backendHost,omitempty"`
	BackendPort int      `json:"backendPort"`
	Ssl         bool     `json:"ssl"`
	Port        int      `json:"port,omitempty"`
	AllowedIps  []string `json:"allowedIps"`
	BlockedIps  []string `json:"blockedIps"`
}

type HttpService struct {
	ID int64 `json:"id"`
	HttpServiceParams
	NodeId       int64     `json:"nodeId"`
	Node         Node      `json:"node"`
	Enabled      bool      `json:"enabled"`
	PublicAccess string    `json:"publicAccess"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type TcpService struct {
	ID int64 `json:"id"`
	TcpServiceParams
	NodeId       int64     `json:"nodeId"`
	Node         Node      `json:"node"`
	Enabled      bool      `json:"enabled"`
	PublicAccess string    `json:"publicAccess"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type PAT struct {
	ID       int64     `json:"id"`
	Name     string    `json:"name"`
	Token    string    `json:"token"`
	ExpireAt time.Time `json:"expireAt"`
	Revoked  bool      `json:"revoked"`
	NodeId   int64     `json:"nodeId"`
}

type NodeParams struct {
	Name           string `json:"name"`
	Address        string `json:"address"`
	GatewayNetwork string `json:"gatewayNetwork"`
	IsGateway      bool   `json:"isGateway"`
	AllowInternet  bool   `json:"allowInternet"`
}

type Node struct {
	ID int64 `json:"id"`
	NodeParams
	WgInterface  string        `json:"wgInterface"`
	Enabled      bool          `json:"enabled"`
	CreatedAt    time.Time     `json:"created_at"`
	UpdatedAt    time.Time     `json:"updated_at"`
	HttpServices []HttpService `json:"httpServices"`
	TcpServices  []TcpService  `json:"tcpServices"`
	Token        string        `json:"token"`
}

type NodeInfo struct {
	Node
	ClientIp                 string `json:"clientIp"`
	LatestHandshakeTimestamp int64  `json:"latestHandshakeTimestamp"`
	TransferRx               int64  `json:"transferRx"`
	TransferTx               int64  `json:"transferTx"`
	Status                   string `json:"status"`
}

type PeerEndpoint struct {
	Url  string `json:"url"`
	Host string `json:"host"`
	Port int    `json:"port"`
}

type PeerConfig struct {
	PublicKey                   string       `json:"publicKey"`
	PresharedKey                string       `json:"presharedKey"`
	Endpoint                    PeerEndpoint `json:"endpoint"`
	PersistentKeepaliveInterval int          `json:"persistentKeepalive"`
	AllowedIPs                  []string     `json:"allowedIPs"`
}

type WGConfig struct {
	PrivateKey string     `json:"privateKey"`
	Address    string     `json:"address"`
	PostUP     []string   `json:"postUp"`
	PostDown   []string   `json:"postDown"`
	Peer       PeerConfig `json:"peer"`
}

type ApiConfig struct {
	VPN_HOST                string
	TCP_SERVICES_PORT_RANGE string
}

type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

type ValidationErrors struct {
	Params []ValidationError `json:"params"`
	Body   []ValidationError `json:"body"`
}

type UnprocessableRequest struct {
	Message string           `json:"message"`
	Errors  ValidationErrors `json:"errors"`
}

type BadRequest struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

func AdminLogin(server string, username string, password string) (string, error) {
	body, _ := json.Marshal(AdminCredentials{
		Username: username,
		Password: password,
	})

	resp := requestApi(apiRequest{Server: server, Method: "POST", Path: "/auth/login", Body: body})

	if resp != nil {
		response := adminLoginResponse{}

		err := json.Unmarshal(resp, &response)

		if err != nil {
			return "", err
		}

		if response.Token != "" {
			return response.Token, nil
		}
	}

	return "", errors.New("authentication failed")
}

func ConfigureNode(server string, token string, node NodeParams) (Node, error) {
	body, _ := json.Marshal(node)

	resp := requestApi(apiRequest{Server: server, Token: token, Method: "POST", Path: "/nodes", Body: body})

	if resp != nil {
		node := Node{}

		err := json.Unmarshal(resp, &node)

		if err != nil {
			fmt.Println(err.Error())
		}

		if node.Token != "" {
			SaveServerConfig(server, node.Token)
			return node, nil
		}
	}

	return Node{}, errors.New("unable to retrieve node configuration")
}

func GetNode() NodeInfo {
	resp := requestApi(apiRequest{Method: "GET", Path: "/cli/node"})

	if resp != nil {
		node := NodeInfo{}

		err := json.Unmarshal(resp, &node)

		if err != nil {
			fmt.Println(err.Error())
		}

		return node
	}

	return NodeInfo{}
}

func GetServices() []HttpService {
	fmt.Println("Getting HTTP services...")
	resp := requestApi(apiRequest{Method: "GET", Path: "/cli/services/http"})

	if resp != nil {
		services := []HttpService{}

		err := json.Unmarshal(resp, &services)

		if err != nil {
			fmt.Println(err.Error())
		}

		return services
	}

	return []HttpService{}
}

func GetTcpServices() []TcpService {
	fmt.Println("Getting TCP services...")
	resp := requestApi(apiRequest{Method: "GET", Path: "/cli/services/tcp"})

	if resp != nil {
		services := []TcpService{}

		err := json.Unmarshal(resp, &services)

		if err != nil {
			fmt.Println(err.Error())
		}

		return services
	}

	return []TcpService{}
}

func GetNodeConfig() string {
	// resp := requestApi("GET", "/cli/config", nil)
	resp := requestApi(apiRequest{Method: "GET", Path: "/cli/config"})

	if resp != nil {
		return strings.ReplaceAll(strings.Trim(string(resp), "\""), "\\n", "\n")
	}

	return ""
}

func GetApiConfig() ApiConfig {
	resp := requestApi(apiRequest{Method: "GET", Path: "/config", Timeout: 5})

	if resp != nil {
		config := ApiConfig{}

		err := json.Unmarshal(resp, &config)

		if err != nil {
			fmt.Println(err.Error())
		}

		return config
	}

	return ApiConfig{}
}

func GetNodeWGConfig() WGConfig {
	// resp := requestApi("GET", "/cli/wgconfig", nil)
	resp := requestApi(apiRequest{Method: "GET", Path: "/cli/wgconfig"})

	if resp != nil {
		config := WGConfig{}

		err := json.Unmarshal(resp, &config)

		if err != nil {
			fmt.Println(err.Error())
		}

		return config
	}

	return WGConfig{}
}

func RegenerateKeys() {
	resp := requestApi(apiRequest{Method: "PATCH", Path: "/cli/regenerate"})

	if resp != nil {
		node := Node{}

		err := json.Unmarshal(resp, &node)

		if err != nil {
			fmt.Println(err.Error())
		}

		config := getConfig()

		if node.Token != "" {
			SaveServerConfig(config.Server.Url, node.Token)
			Connect(ConnectionConfig{})
		} else {
			fmt.Println("Error getting new keys.")
			return
		}
	}
}

func ExposeHTTP(service HttpServiceParams, node NodeInfo) {
	body, _ := json.Marshal(service)

	// resp := requestApi("POST", "/cli/expose/http", body)
	resp := requestApi(apiRequest{Method: "POST", Path: "/cli/expose/http", Body: body})

	if resp != nil {
		createdService := HttpService{}

		err := json.Unmarshal(resp, &createdService)

		if err != nil {
			fmt.Printf("❌ Unable to expose HTTP service: %v\n", err)
		}

		fmt.Println("New Service Available")

		PrintHttpServices([]HttpService{createdService}, node.IsGateway)
	}
}

func ExposeTCP(service TcpServiceParams, node NodeInfo) {
	body, _ := json.Marshal(service)

	resp := requestApi(apiRequest{Method: "POST", Path: "/cli/expose/tcp", Body: body})

	if resp != nil {
		createdService := TcpService{}

		err := json.Unmarshal(resp, &createdService)

		if err != nil {
			fmt.Printf("❌ Unable to expose TCP service: %v\n", err)
		}

		fmt.Println("New Service Available")

		PrintTcpServices([]TcpService{createdService}, node.IsGateway)
	}
}

func DisableServiceByType(serviceType string, id string) {
	resp := requestApi(apiRequest{Method: "PATCH", Path: "/cli/services/" + serviceType + "/" + id + "/disable"})

	if resp != nil {
		if serviceType == "http" {
			service := HttpService{}

			err := json.Unmarshal(resp, &service)

			if err != nil {
				fmt.Printf("❌ Unable to disable service: %v\n", err)
			}

			fmt.Println("Service Disabled Successfully!")

			PrintHttpServices([]HttpService{service}, service.BackendHost != "")
		}
		if serviceType == "tcp" {
			service := TcpService{}

			err := json.Unmarshal(resp, &service)

			if err != nil {
				fmt.Printf("❌ Unable to disable service: %v\n", err)
			}

			fmt.Println("Service Disabled Successfully!")

			PrintTcpServices([]TcpService{service}, service.BackendHost != "")
		}
	}
}

func EnableServiceByType(serviceType string, id string) {
	resp := requestApi(apiRequest{Method: "PATCH", Path: "/cli/services/" + serviceType + "/" + id + "/enable"})

	if resp != nil {
		if serviceType == "http" {
			service := HttpService{}

			err := json.Unmarshal(resp, &service)

			if err != nil {
				fmt.Printf("❌ Unable to enable service: %v\n", err)
			}

			fmt.Println("Service Enabled Successfully!")

			PrintHttpServices([]HttpService{service}, service.BackendHost != "")
		}
		if serviceType == "tcp" {
			service := TcpService{}

			err := json.Unmarshal(resp, &service)

			if err != nil {
				fmt.Printf("❌ Unable to disabled service: %v\n", err)
			}

			fmt.Println("Service Enabled Successfully!")

			PrintTcpServices([]TcpService{service}, service.BackendHost != "")
		}
	}
}

func PrintHttpServices(services []HttpService, IsGateway bool) {
	for _, svc := range services {
		var target = ""

		var enabled = "✅"

		if !svc.Enabled {
			enabled = "❌"
		}

		if IsGateway {
			target = svc.BackendProto + "://" + svc.BackendHost + ":" + strconv.Itoa(svc.BackendPort)
		} else {
			target = svc.BackendProto + "://localhost:" + strconv.Itoa(svc.BackendPort)
		}

		fmt.Printf("- %s %s %s [HTTP] → %s → %s\n",
			strconv.Itoa(int(svc.ID)), enabled, svc.Name, svc.PublicAccess, target)
	}
}

func PrintTcpServices(services []TcpService, IsGateway bool) {
	for _, svc := range services {
		var target = ""

		var enabled = "✅"

		if !svc.Enabled {
			enabled = "❌"
		}

		proto := strings.ToUpper(svc.Proto)
		if svc.Ssl {
			proto += "/SSL"
		}

		if IsGateway {
			target = svc.Proto + "://" + svc.BackendHost + ":" + strconv.Itoa(svc.BackendPort)
		} else {
			target = svc.Proto + "://localhost:" + strconv.Itoa(svc.BackendPort)
		}

		fmt.Printf("- %s %s %s [%s] → %s://%s → %s\n",
			strconv.Itoa(int(svc.ID)), enabled, svc.Name, proto, svc.Proto, svc.PublicAccess, target)
	}
}

func requestApi(request apiRequest) []byte {
	config := getConfig()

	server := config.Server.Url

	if request.Server != "" {
		server = request.Server
	}

	base, _ := url.Parse(server)
	base.Path = path.Join(base.Path, path.Join(config.Server.Path, "/api", request.Path))

	timeout := 20

	if request.Timeout > 0 {
		timeout = request.Timeout
	}

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{
		Transport: tr,
		Timeout:   time.Second * time.Duration(timeout),
	}

	// fmt.Println("Requesting", base.String())

	req, err := http.NewRequest(request.Method, base.String(), bytes.NewBuffer(request.Body))

	if err != nil {
		fmt.Printf("❌ Unable to perform request: %v\n", err)
	}

	var token string

	if request.Token != "" {
		token = request.Token
	} else {
		token = config.Server.Token
	}

	req.Header.Add("Accept", "application/json")
	req.Header.Add("Content-Type", "application/json")
	req.Header.Set("User-Agent", "wiredoor-cli/" + version.Version)

	if token != "" {
		req.Header.Add("Authorization", "Bearer "+token)
	}

	resp, err := client.Do(req)

	if err != nil {
		fmt.Printf("❌ Request failed: %v\n", err)
		return nil
	}

	defer resp.Body.Close()
	bodyBytes, err := io.ReadAll(resp.Body)

	if resp.StatusCode == 400 {
		errorRes := BadRequest{}

		_ = json.Unmarshal(bodyBytes, &errorRes)

		fmt.Println("Bad Request: ", errorRes.Message)
		return nil
	}

	if resp.StatusCode == 403 || resp.StatusCode == 401 {
		fmt.Println("Error: Invalid authentication token")
		return nil
	}

	if resp.StatusCode == 422 {
		errorRes := UnprocessableRequest{}

		_ = json.Unmarshal(bodyBytes, &errorRes)

		if len(errorRes.Errors.Body) > 0 {
			for _, v := range errorRes.Errors.Body {
				fmt.Println("Error - "+v.Field+":", v.Message)
			}
		}

		return nil
	}

	if resp.StatusCode >= 500 {
		fmt.Println("Error: Unknown Wiredoor server error")
		return nil
	}
	if err != nil {
		fmt.Printf("❌ Unable to read body response: %v\n", err)
	}

	return bodyBytes
}
