package wiredoor

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/wiredoor/wiredoor-cli/utils"
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

type EnableRequest struct {
	ID          string
	ServiceType string
	Ttl         string
}

type EnableParams struct {
	Ttl string `json:"ttl"`
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
	Ttl          string   `json:"ttl"`
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
	Ttl         string   `json:"ttl"`
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

type GatewayNetwork struct {
	Interface string `json:"interface"`
	Subnet    string `json:"subnet"`
}

type NodeParams struct {
	Name            string           `json:"name"`
	Address         string           `json:"address"`
	GatewayNetworks []GatewayNetwork `json:"gatewayNetworks"`
	IsGateway       bool             `json:"isGateway"`
	AllowInternet   bool             `json:"allowInternet"`
}

type Node struct {
	ID int64 `json:"id"`
	NodeParams
	GatewayNetwork string        `json:"gatewayNetwork"`
	WgInterface    string        `json:"wgInterface"`
	Enabled        bool          `json:"enabled"`
	CreatedAt      time.Time     `json:"created_at"`
	UpdatedAt      time.Time     `json:"updated_at"`
	HttpServices   []HttpService `json:"httpServices"`
	TcpServices    []TcpService  `json:"tcpServices"`
	Token          string        `json:"token"`
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

type UpdateGatewayParams struct {
	GatewayInterface string `json:"gatewayInterface"`
	GatewayNetwork   string `json:"gatewayNetwork"`
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
			utils.Terminal().Errorf("Unmarshal: %v", err)
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
			utils.Terminal().Errorf("Unable to retrieve node information: %v", err)
		}

		return node
	}

	return NodeInfo{}
}

func GetServices() []HttpService {
	// !!!NOT USED ???
	utils.Terminal().Println("Getting HTTP services...")
	resp := requestApi(apiRequest{Method: "GET", Path: "/cli/services/http"})

	if resp != nil {
		services := []HttpService{}

		err := json.Unmarshal(resp, &services)

		if err != nil {
			utils.Terminal().Errorf("Unable to retrieve HTTP services: %v", err)
		}

		return services
	}

	return []HttpService{}
}

func GetTcpServices() []TcpService {
	// !!! NOT USED
	utils.Terminal().Println("Getting TCP services...")
	resp := requestApi(apiRequest{Method: "GET", Path: "/cli/services/tcp"})

	if resp != nil {
		services := []TcpService{}

		err := json.Unmarshal(resp, &services)

		if err != nil {
			utils.Terminal().Errorf("Unable to retrieve TCP services: %v", err)
		}

		return services
	}

	return []TcpService{}
}

func GetNodeConfig() string {
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
			utils.Terminal().Errorf("Unable to retrieve API configuration: %v", err)
		}

		return config
	}

	return ApiConfig{}
}

func GetNodeWGConfig() WGConfig {
	resp := requestApi(apiRequest{Method: "GET", Path: "/cli/wgconfig"})

	if resp != nil {
		config := WGConfig{}

		err := json.Unmarshal(resp, &config)

		if err != nil {
			utils.Terminal().Errorf("Unable to retrieve WireGuard configuration: %v", err)
		}

		return config
	}

	return WGConfig{}
}

func RegenerateKeys() error {
	resp := requestApi(apiRequest{Method: "PATCH", Path: "/cli/regenerate"})

	if resp != nil {
		node := Node{}

		err := json.Unmarshal(resp, &node)

		if err != nil {
			utils.Terminal().Errorf("Unable to regenerate keys: %v", err)
		}

		config := getConfig()

		if node.Token != "" {
			SaveServerConfig(config.Server.Url, node.Token)
			Connect(ConnectionConfig{})
		} else {
			if err != nil {
				utils.Terminal().Errorf("Unable to regenerate keys. Warning node config error:%v", err)
				return err
			}
			utils.Terminal().Errorf("Unable to regenerate keys. No token received.")
			return errors.New("unable to regenerate keys. No token received")
		}
	}
	return nil
}

func ExposeHTTP(service HttpServiceParams, node NodeInfo) {
	body, _ := json.Marshal(service)

	resp := requestApi(apiRequest{Method: "POST", Path: "/cli/expose/http", Body: body})

	if resp != nil {
		createdService := HttpService{}

		err := json.Unmarshal(resp, &createdService)

		if err != nil {
			utils.Terminal().Errorf("Unable to expose HTTP service: %v", err)
			return
		}

		utils.Terminal().Section("New Service Available")

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
			utils.Terminal().Errorf("Unable to expose TCP service: %v", err)
			return
		}

		utils.Terminal().Section("New Service Available")

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
				utils.Terminal().Errorf("Unable to disable service: %v", err)
			}

			utils.Terminal().Section("Service Disabled Successfully!")

			PrintHttpServices([]HttpService{service}, service.BackendHost != "")
		}
		if serviceType == "tcp" {
			service := TcpService{}

			err := json.Unmarshal(resp, &service)

			if err != nil {
				utils.Terminal().Errorf("Unable to disable service: %v", err)
			}

			utils.Terminal().Section("Service Disabled Successfully!")

			PrintTcpServices([]TcpService{service}, service.BackendHost != "")
		}
	}
}

func EnableServiceByType(params EnableRequest) {
	var body []byte

	if params.Ttl != "" {
		body, _ = json.Marshal(EnableParams{Ttl: params.Ttl})
	}

	resp := requestApi(apiRequest{Method: "PATCH", Path: "/cli/services/" + params.ServiceType + "/" + params.ID + "/enable", Body: body})

	if resp != nil {
		if params.ServiceType == "http" {
			service := HttpService{}

			err := json.Unmarshal(resp, &service)

			if err != nil {
				utils.Terminal().Errorf("Unable to enable service: %v", err)
			}

			utils.Terminal().Section("Service Enabled Successfully!")

			PrintHttpServices([]HttpService{service}, service.BackendHost != "")
		}
		if params.ServiceType == "tcp" {
			service := TcpService{}

			err := json.Unmarshal(resp, &service)

			if err != nil {
				utils.Terminal().Errorf("Unable to enable service: %v", err)
			}

			utils.Terminal().Section("Service Enabled Successfully!")

			PrintTcpServices([]TcpService{service}, service.BackendHost != "")
		}
	}
}

func UpdateGatewaySubnet(network GatewayNetwork) {
	body, _ := json.Marshal(UpdateGatewayParams{GatewayNetwork: network.Subnet, GatewayInterface: network.Interface})

	resp := requestApi(apiRequest{Method: "PATCH", Path: "/cli/node/gateway", Body: body})

	if resp != nil {
		node := NodeInfo{}

		err := json.Unmarshal(resp, &node)

		if err != nil {
			utils.Terminal().Errorf("Unable to update gateway subnet: %v", err)
		}

		utils.Terminal().Section("Subnet successfully updated to: " + node.GatewayNetwork)
	}
}

func PrintHttpServices(services []HttpService, isGateway bool) {
	utils.Terminal().Section("  HTTP:")
	if len(services) == 0 {
		utils.Terminal().KV("HTTP", "none")
		return
	}

	rows := make([][]string, 0, len(services))
	for _, svc := range services {
		enabled := "disabled"
		if svc.Enabled {
			enabled = "enabled"
		}

		target := buildHttpTarget(svc, isGateway)

		rows = append(rows, []string{
			strconv.FormatInt(int64(svc.ID), 10),
			enabled,
			svc.Name,
			svc.PublicAccess,
			target,
		})
	}

	utils.Terminal().Table([]string{"ID", "STATE", "NAME", "PUBLIC", "TARGET"}, rows)
}

func buildHttpTarget(svc HttpService, isGateway bool) string {
	port := strconv.Itoa(svc.BackendPort)
	if isGateway {
		return svc.BackendProto + "://" + svc.BackendHost + ":" + port
	}
	return svc.BackendProto + "://localhost:" + port
}

func PrintTcpServices(services []TcpService, isGateway bool) {
	utils.Terminal().Section("  TCP:")

	if len(services) == 0 {
		utils.Terminal().KV("TCP", "none")
		return
	}

	rows := make([][]string, 0, len(services))
	for _, svc := range services {
		state := "disabled"
		if svc.Enabled {
			state = "enabled"
		}

		proto := strings.ToUpper(svc.Proto)
		if svc.Ssl {
			proto += "/SSL"
		}

		target := buildTcpTarget(svc, isGateway)

		rows = append(rows, []string{
			strconv.FormatInt(int64(svc.ID), 10),
			state,
			svc.Name,
			proto,
			svc.PublicAccess,
			target,
		})
	}

	utils.Terminal().Table([]string{"ID", "STATE", "NAME", "PROTO", "PUBLIC", "TARGET"}, rows)
}

func buildTcpTarget(svc TcpService, isGateway bool) string {
	port := strconv.Itoa(svc.BackendPort)
	if isGateway {
		return svc.Proto + "://" + svc.BackendHost + ":" + port
	}
	return svc.Proto + "://localhost:" + port
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
		utils.Terminal().Errorf("Unable to perform request: %v", err)
	}

	var token string

	if request.Token != "" {
		token = request.Token
	} else {
		token = config.Server.Token
	}

	req.Header.Add("Accept", "application/json")
	req.Header.Add("Content-Type", "application/json")
	req.Header.Set("User-Agent", "wiredoor-cli/"+version.Version)

	if token != "" {
		req.Header.Add("Authorization", "Bearer "+token)
	}

	resp, err := client.Do(req)

	if err != nil {
		utils.Terminal().Errorf("Request failed: %v", err)
		return nil
	}

	defer resp.Body.Close()
	bodyBytes, err := io.ReadAll(resp.Body)

	if resp.StatusCode == 400 {
		errorRes := BadRequest{}

		_ = json.Unmarshal(bodyBytes, &errorRes)

		utils.Terminal().Errorf("Bad Request: %s", errorRes.Message)
		return nil
	}

	if resp.StatusCode == 403 || resp.StatusCode == 401 {
		utils.Terminal().Errorf("Invalid authentication token")
		return nil
	}

	if resp.StatusCode == 404 {
		utils.Terminal().Errorf("Server not found. Please check your server URL configuration.")
		return nil
	}

	if resp.StatusCode == 422 {
		errorRes := UnprocessableRequest{}

		_ = json.Unmarshal(bodyBytes, &errorRes)

		if len(errorRes.Errors.Body) > 0 {
			for _, v := range errorRes.Errors.Body {
				utils.Terminal().Errorf(" -> %s: %s", v.Field, v.Message)
			}
		}

		return nil
	}

	if resp.StatusCode >= 500 {
		utils.Terminal().Errorf("Unknown Wiredoor server error")
		return nil
	}

	if !strings.Contains(strings.ToLower(resp.Header.Get("Content-Type")), "application/json") {
		utils.Terminal().Errorf("Unexpected response format: %s", resp.Header.Get("Content-Type"))
		return nil
	}

	if err != nil {
		utils.Terminal().Errorf("Unable to read body response: %v", err)
	}

	return bodyBytes
}
