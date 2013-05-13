package mconn

import (
	"errors"
	"fmt"
	"net/rpc"
)

type Args struct {
	Pid int
}

type Process struct {
	Name string
	Pid  int
	Cpu  float64
	Mem  float64
}

type Mconn struct {
	DialServer string
	State      string
	client     *rpc.Client
}

type RPC_Server struct {
}

func (conn *Mconn) GetProInfo(proInfo *map[string]*Process) error {
	if conn.client == nil {
		return errors.New("mconn:conn is not dialed")
	}
	args := &Args{}
	err := conn.client.Call("RPC_Server.GetProcessInfo", args, proInfo)
	if err != nil {
		fmt.Println(err.Error())
		return err
	}

	// for _, v := range *proInfo {
	// 	fmt.Println(v.Cpu)
	// 	fmt.Println(v.Pid)
	// }
	return nil
}

func (conn *Mconn) Dial() error {
	if conn.DialServer == "" {
		return errors.New("no server to dial.")
	}

	var err error
	conn.client, err = rpc.Dial("tcp", conn.DialServer)
	if err != nil {
		return err
	}
	conn.State = "CONNECTED"
	return nil
}

func (conn *Mconn) IsDialed() bool {
	return conn.State == "CONNECTED"
}

func (conn *Mconn) Close() {
	conn.client.Close()
	conn.State = "CLOSE"
}

// func FindRpcClient(pid int) *rpc.Client {
// 	fmt.Println(RPC_Clients)
// 	for _, v := range RPC_Clients {
// 		fmt.Println(len(v.DialServer.Pros))
// 		for k, pro := range v.DialServer.Pros {
// 			fmt.Println(k, pro)
// 			fmt.Println(pro.Pid)
// 			if pid == pro.Pid {
// 				fmt.Println("find pid ", pro.Pid)
// 				fmt.Println(v.client)
// 				return v.client
// 			}
// 		}
// 	}
// 	return nil
// }
