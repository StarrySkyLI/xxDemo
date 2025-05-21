package main

import (
	"fmt"
	"xiexinDemo/apis"
	"xiexinDemo/core"
	"xiexinDemo/myzinx/ziface"
	"xiexinDemo/myzinx/znet"
)

// 当前客户端建立连接后的hook函数
func OnConnectionAdd(conn ziface.IConnection) {
	//创建player
	player := core.NewPlayer(conn)

	//给客户端发送MsgID：1的消息 :同步当前player的id给客户端
	player.SyncPid()
	//给客户端发送MsgID：200的消息：同步初始化位置
	player.BroadCastStartPosition()
	//将当前新上线玩家添加到worldManager中
	core.WorldMgrObj.AddPlayer(player)
	//将该连接绑定一个pid玩家ID的属性
	conn.Setproperty("pid", player.Pid)
	//同步周边玩家，告知当前玩家上线，广播当前玩家位置
	player.SynvSurrounding()

	fmt.Println("===>player pid ", player.Pid, " is arrived<=====")
}

// 当前客户端断开连接后的hook函数
func OnConnectionLost(conn ziface.IConnection) {
	//获取当前连接的绑定的Pid
	pid, _ := conn.Getproperty("pid")

	//根据pid获取对应的玩家对象
	player := core.WorldMgrObj.GetPlayerByPid(pid.(int32))

	//触发玩家下线业务
	player.Offline()

	fmt.Println("====> Player ", pid, " left =====")
}
func main() {
	//创建zinx server句柄
	s := znet.NewServer("MMO Game ")

	//连接创建和销毁的HOOK钩子函数
	s.SetOnConnStart(OnConnectionAdd)
	s.SetOnConnStop(OnConnectionLost)

	//注册一些路由业务
	s.AddRouter(2, &apis.WorldChatApi{})
	s.AddRouter(3, &apis.MoveApi{})

	//启动服务
	s.Serve()

}
