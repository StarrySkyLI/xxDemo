package apis

import (
	"fmt"
	"google.golang.org/protobuf/proto"
	"xiexinDemo/core"
	"xiexinDemo/myzinx/ziface"
	"xiexinDemo/myzinx/znet"
	"xiexinDemo/pb"
)

//世界聊天 路由业务

type WorldChatApi struct {
	znet.BaseRouter
}

func (wc *WorldChatApi) Handle(requet ziface.IRequest) {
	//1 解析客户端传递进来的proto协议
	proto_msg := &pb.Talk{}
	err := proto.Unmarshal(requet.GetData(), proto_msg)
	if err != nil {
		fmt.Println("Talk Unmarshal error ", err)
		return
	}

	//2 当前的聊天数据是属于谁发送的
	pid, err := requet.GetConnection().Getproperty("pid")
	//3 根据pid得到当前玩家对应的player对象
	player := core.WorldMgrObj.GetPlayerByPid(pid.(int32))

	//4 将这个消息广播给其他全部在线的玩家
	player.Talk(proto_msg.Content)

}
