package main

import (
	"code.google.com/p/go.net/websocket"
	"config"
	"encoding/json"
	"errors"
	"fmt"
	"goMonitor/mconn"
	"html/template"
	"log"
	"net/http"
	"net/smtp"
	"os"
	"path"
	"runtime"
	"time"
)

//Protocol contain three forms of message that client and Monitor server can transfer
//information,they are 'start' 'stop' 'restart' 'stop all' 'restart all'

type Protocol struct {
	Operation string
	Servers   []interface{}
}

type SendInfo struct {
	ServerIp string
	ProInfo  []map[string]*mconn.Process
}

type ProInfo map[string][]map[string]*mconn.Process

type DialServer struct {
	ServerIp string
	Conns    []*mconn.Mconn
}

type WSConn struct {
	Conn    *websocket.Conn
	Servers []string
	bSend   bool
}

const VERSION = "0.2.0"

var (
	MonServers      map[string]*DialServer
	CONFIG_PATH     string
	APP_PATH        string
	VIEW_PATH       string
	user2psd        map[string]string = make(map[string]string)
	Operations      []string
	CmdChan         chan bool
	DialServersChan chan []*DialServer
	QuitWSChan      chan bool
	EndMonChan      chan bool
	DataFlowState   bool
	SEND_INFO_DELAY time.Duration
	ConnNum         int
	WSConnChan      chan *WSConn
	SendInfoChan    chan *SendInfo
)

func initData() {
	runtime.GOMAXPROCS(2)
	APP_PATH, _ = os.Getwd()
	CONFIG_PATH = path.Join(APP_PATH, "conf")
	VIEW_PATH = path.Join(APP_PATH, "view")

	MonServers = make(map[string]*DialServer)
	CmdChan = make(chan bool)
	DialServersChan = make(chan []*DialServer)
	QuitWSChan = make(chan bool)
	EndMonChan = make(chan bool)
	WSConnChan = make(chan *WSConn)
	SendInfoChan = make(chan *SendInfo)
	DataFlowState = false
	//init user/password data
	user2psd["u1"] = "p1"
	user2psd["u2"] = "p2"
	user2psd["u3"] = "p3"

	SEND_INFO_DELAY = 100 * time.Millisecond
	//init processes which will be monitored
	//MonitorProInfo = &ProcessInfo{ColPname: "pname", ColPid: "pid", ColCpu: "%cpu", ColMem: "%mem",
	//	Pros: make(map[string]*Process)}

	InitConfig()
	go dataFlow()
}

//func filter([]string, func ())

func SendMail() {
	// Set up authentication information.
	auth := smtp.PlainAuth(
		"",             // identity
		"test",         // user name
		"123456",       // password
		"smtp.163.com", // host
	)
	// Connect to the server, authenticate, set the sender and recipient,
	// and send the email all in one step.
	err := smtp.SendMail(
		"smtp.163.com:25",                 // addr eg: mail.example.com:25
		auth,                              // auth
		"test@163.com",                    // from
		[]string{"test2@163.com"},         // to
		[]byte("This is the email body."), // msg
	)
	if err != nil {
		log.Fatal(err)
	}
}

func viewFilePath(file string) string {
	return path.Join(VIEW_PATH, file)
}

func confFilePath(file string) string {
	return path.Join(CONFIG_PATH, file)
}

func loginCheck(username string, password string) (err error) {
	bInvalidUsername := false
	bInvalidPassword := false
	err = nil
	for k, v := range user2psd {
		if username == k {
			bInvalidUsername = true
			if password == v {
				bInvalidPassword = true
				break
			} else {
				bInvalidPassword = false
			}
		}
	}
	if bInvalidUsername == false {
		fmt.Println(username, " doesn't exists.")
		err = errors.New(username + " doesn't exists.")
		//return UNAME_NO_EXIST
	} else if bInvalidPassword == false {
		fmt.Println("wrong password")
		err = errors.New("wrong password")
		//return PASSWORD_WRONG
	}
	return err
}

func send(ws *websocket.Conn, msg interface{}) error {
	err := websocket.JSON.Send(ws, msg)
	return err
}

func recv(ws *websocket.Conn) (*Protocol, error) {
	var p Protocol
	fmt.Println("receiving message...")
	err := websocket.JSON.Receive(ws, &p)
	if err != nil {
		return &p, err
	}
	return &p, nil
}

func WS_Handler(ws *websocket.Conn) {
	ConnNum++
	defer func() {
		// if DataFlowState == true {
		// 	QuitWSChan <- true
		// }
		ws.Close()
		fmt.Println("websocket closed")
	}()

	for {
		clientData, err := recv(ws)
		if err != nil {
			fmt.Println("recv error:" + err.Error())
			break
		}
		//fmt.Println(clientData)
		// fmt.Println(reflect.TypeOf(clientData.Servers))
		// for k, v := range clientData.Servers {
		// 	fmt.Println(k, v)
		// }
		if DealClientMsg(ws, clientData) == false {
			break
		}
	}
}

func DialAllServer(wsConn *WSConn, servers []interface{}) {
	curServer := make([]*DialServer, 0)
	for i := 0; i < len(servers); i++ {
		server := servers[i].(map[string]interface{})
		for _, v := range server {
			ip := v.(string)
			//fmt.Println(ip)
			dialServer, ok := MonServers[ip]
			if !ok {
				continue
			}
			wsConn.Servers = append(wsConn.Servers, ip)
			s := DialServer{}
			s.ServerIp = ip
			s.Conns = make([]*mconn.Mconn, 0)
			for _, conn := range dialServer.Conns {
				if conn.IsDialed() {
					continue
				}
				//fmt.Println("dial server:" + conn.DialServer)
				err := conn.Dial()
				if err != nil {
					fmt.Println("dialAllServer error: " + err.Error())
					continue
				}
				s.Conns = append(s.Conns, conn)
			}
			curServer = append(curServer, &s)
		}
	}
	// fmt.Println("len of s: ", len(curServer))
	// for _, item := range curServer {
	// 	fmt.Println(item.ServerIp)
	// 	for _, v := range item.Conns {
	// 		fmt.Println(v.DialServer)
	// 		fmt.Println(v.State)
	// 	}
	// }
	WSConnChan <- wsConn
	DialServersChan <- curServer
	fmt.Println("dial servers done")
}

func DealClientMsg(ws *websocket.Conn, clientData *Protocol) bool {
	switch clientData.Operation {
	case "start monitor":

		DialAllServer(&WSConn{ws, make([]string, 0), true}, clientData.Servers)
		return true
		//EndMonChan <- true
	case "end monitor":
		WSConnChan <- &WSConn{ws, make([]string, 0), false}
		// if len(clientData.Servers) == 0 {
		// 	return
		// }
		//EndMonChan <- false
		return true
	case "stop":
		return true
		// var pid int
		// client := FindRpcClient(pid)
		// if client == nil {
		// 	fmt.Println("no client conn")
		// 	break
		// }
		// pro := &Process{}
		// args := &Args{pid}
		// err := client.Call("ProcessInfo.KillProcess", args, pro)
		// if err != nil {
		// 	panic(err)
		// }
		// fmt.Println(pro.Pid)
	}
	return false
}

func PrintInfo(info []*SendInfo) {
	type SendInfo struct {
		ServerIp string
		ProInfo  []map[string]*mconn.Process
	}
	for _, v := range info {
		for _, p := range v.ProInfo {
			for _, c := range p {
				fmt.Println(c.Cpu)
				fmt.Println(c.Mem)
				fmt.Println(c.Name)
				fmt.Println(c.Pid)
			}
		}
	}
}

func dataFlow() {
	//fmt.Println("conn num:", ConnNum)
	DataFlowState = true
	dialServers := make([]*DialServer, 0)
	conns := make(map[*websocket.Conn]*WSConn)
	for {
		select {
		case server := <-DialServersChan:
			dialServers = append(dialServers, server...)
		case conn := <-WSConnChan:
			conns[conn.Conn] = conn

		default:
			sendInfo := getProFromServers(dialServers)
			// for ip, v := range *sendInfo {
			// 	fmt.Println("ip: " + ip)
			// 	for _, item := range v {
			// 		for _, va := range item {
			// 			fmt.Println(va.Pid)
			// 		}
			// 	}
			// }
			for ws, WSConnection := range conns {
				if !WSConnection.bSend {
					continue
				}
				info := make([]*SendInfo, 0)
				for _, serverIp := range WSConnection.Servers {
					for _, v := range sendInfo {
						if v.ServerIp == serverIp {
							info = append(info, v)
						}
					}
				}
				//PrintInfo(info)
				msg, err := json.Marshal(info)
				if err != nil {
					panic(err)
				}
				err = send(WSConnection.Conn, string(msg))
				if err != nil {
					fmt.Println("send error: " + err.Error())
					// if connIndex+1 < len(conns) {
					// 	copy(conns[connIndex:], conns[connIndex+1:])
					// }
					// conns = conns[:len(conns)-1]
					delete(conns, ws)
				}
			}
			time.Sleep(SEND_INFO_DELAY)
		}
	}
}

func getProFromServers(dialServers []*DialServer) []*SendInfo {
	//fmt.Println("len of conns: ", len(dialServers))
	result := make([]*SendInfo, 0)

	for i := 0; i < len(dialServers); i++ {
		s := dialServers[i]
		Si := &SendInfo{}
		Si.ProInfo = make([]map[string]*mconn.Process, 0)
		Si.ServerIp = s.ServerIp
		pro := make(map[string]*mconn.Process)

		for _, conn := range s.Conns {
			conn.GetProInfo(&pro)
			Si.ProInfo = append(Si.ProInfo, pro)
		}
		result = append(result, Si)
	}
	return result
}

func InitConfig() {
	serverListConf, err := config.OpenIniFile(confFilePath("serverlist.conf"))
	if err != nil {
		panic(err)
	}

	if err := serverListConf.Parse(); err != nil {
		panic(err)
	}
	for ip, servers := range serverListConf.Sections {
		conns := make([]*mconn.Mconn, 0)
		MonServer := &DialServer{ip, conns}
		for _, node := range servers.Nodes {
			conn := &mconn.Mconn{}
			conn.DialServer = node.Value
			MonServer.Conns = append(MonServer.Conns, conn)
		}
		MonServers[servers.Name] = MonServer
	}

	// for _, v := range MonServers {
	// 	fmt.Println(v.ServerIp)
	// 	for _, c := range v.Conns {
	// 		fmt.Println(c.DialServer)
	// 	}
	// }
	operationConf, err := config.OpenIniFile(confFilePath("operations.conf"))
	if err != nil {
		panic(err)
	}

	if err := operationConf.Parse(); err != nil {
		panic(err)
	}
	sec := operationConf.GetSection("Operations")
	for _, operation := range sec.Nodes {
		Operations = append(Operations, operation.Value)
	}
}

func monitor(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		t, _ := template.ParseFiles(viewFilePath("login_fail.html"))
		t.Execute(w, "You Need to sign in!")
	} else {
		r.ParseForm()
		username := template.HTMLEscapeString(r.Form.Get("username"))
		password := template.HTMLEscapeString(r.Form.Get("password"))
		fmt.Println(username, password)
		if err := loginCheck(username, password); err != nil {
			t, _ := template.ParseFiles(viewFilePath("login_fail.html"))
			t.Execute(w, err.Error())
		} else {
			fmt.Println("login successful!")
			//t, err := template.ParseFiles(viewFilePath("mon.html"))
			http.ServeFile(w, r, viewFilePath("mon.html"))
			// template.Must(t, err)
			// getProcessInfo(MonitorProInfo)
			// t.Execute(w, nil)
		}
	}
}

func login(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		t, _ := template.ParseFiles(viewFilePath("login.html"))
		t.Execute(w, nil)
	} else {
	}
}

func main() {
	initData()

	fmt.Println("GoMonitor starting up...")
	http.HandleFunc("/login", login)
	http.HandleFunc("/monitor", monitor)
	http.Handle("/processInfo", websocket.Handler(WS_Handler))
	http.ListenAndServe(":9091", nil)
}
