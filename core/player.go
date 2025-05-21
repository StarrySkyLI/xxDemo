package core

import (
	"fmt"
	"google.golang.org/protobuf/proto"
	"math/rand"
	"sync"
	"time"
	"xiexinDemo/myzinx/ziface"
	"xiexinDemo/pb"
)

// 玩家对象
type Player struct {
	//玩家ID
	Pid int32
	//当前玩家用于和客户端的连接
	Conn ziface.IConnection
	X    float32 //平面x坐标
	Y    float32 //高度
	Z    float32 //平面y坐标
	V    float32 //旋转0-360角度
}

// player id 生成器 后面生成数据库
// 应该有登录模块，由数据库查询之后再发入ID等信息
var PidGen int32 = 1
var IDLock sync.Mutex

func NewPlayer(conn ziface.IConnection) *Player {
	//生成玩家id
	IDLock.Lock()
	id := PidGen
	PidGen++
	IDLock.Unlock()

	p := &Player{
		Pid:  id,
		Conn: conn,

		X: float32(160 + rand.Intn(10)), //随机在160坐标点 基于x若干偏移
		Y: 0,
		Z: float32(140 + rand.Intn(20)),
		V: 0,
	}
	return p

}

// 提供一个发送给客户端消息的方法
// 主要是将pb的protobuf数据序列化之后，再调用zinx的SendMsg方法
func (p *Player) SendMsg(msgId uint32, data proto.Message) {
	if p.Conn == nil {
		fmt.Println("connection in player is nil")
		return
	}
	//将proto Msg结构体序列化 转换为2进制
	msg, err := proto.Marshal(data)
	if err != nil {
		fmt.Println()
	}
	if err != nil {
		fmt.Println("Marshal err: ", err)

	}
	//将二进制文件通过zinx框架SendMsg将数据发送给客户端
	if p.Conn == nil {
		fmt.Printf("connection in player %d is nil", p.Pid)
		return
	}
	if err := p.Conn.SendMsg(msgId, msg); err != nil {
		fmt.Println("player send msg err")
		return
	}
	return

}

// 告知客户端玩家pid，同步已经生成的玩家id给客户端
func (p *Player) SyncPid() {
	//组建MsgID:0 的proto数据
	proto_msg := &pb.SyncPid{
		Pid: p.Pid,
	}
	p.SendMsg(1, proto_msg)
}

// 广播玩家自己的出生地点
func (p *Player) BroadCastStartPosition() {
	proto_msg := &pb.BroadCast{
		Pid: p.Pid,
		Tp:  2,
		Data: &pb.BroadCast_P{
			P: &pb.Position{
				X: p.X,
				Y: p.Y,
				Z: p.Z,
				V: p.V,
			},
		},
	}
	p.SendMsg(200, proto_msg)
}

// 玩家广播世界聊天消息
func (p *Player) Talk(content string) {
	//1. 组建MsgId200 proto数据
	msg := &pb.BroadCast{
		Pid: p.Pid,
		Tp:  1, //TP 1 代表聊天广播
		Data: &pb.BroadCast_Content{
			Content: content,
		},
	}

	//2. 得到当前世界所有的在线玩家
	players := WorldMgrObj.GetAllPlayers()

	//3. 向所有的玩家发送MsgId:200消息
	for _, player := range players {
		player.SendMsg(200, msg)
	}
}

// 同步周边玩家，告知当前玩家上线，广播当前玩家位置
func (p *Player) SynvSurrounding() {
	//1 根据自己的位置，获取周围九宫格内的玩家pid
	pids := WorldMgrObj.AoiMgr.GetPidsbyPos(p.X, p.Z)
	players := make([]*Player, 0, len(pids))
	for _, pid := range pids {
		players = append(players, WorldMgrObj.GetPlayerByPid(int32(pid)))
	}

	//2 根据pid得到所有玩家对象
	//2.1 组建MsgID：200 proto数据
	proto_msg := &pb.BroadCast{
		Pid: p.Pid,
		Tp:  2,
		Data: &pb.BroadCast_P{
			P: &pb.Position{
				X: p.X,
				Y: p.Y,
				Z: p.Z,
				V: p.V,
			},
		},
	}
	//2.2 全部周围玩家都向格子的客户端发送200消息，让自己出现在对方视野中
	for _, player := range players {
		player.SendMsg(200, proto_msg)
	}

	//3 将周围的全部玩家的位置消息发送给当前的玩家MsgID：202 客户端（让自己看到其他玩家）
	//3.1 组建MsgID：202 proto数据
	//3.1.1制作一个pb.player slice
	players_proto_msg := make([]*pb.Player, 0, len(players))
	for _, player := range players {
		//制作一个messager player
		p := &pb.Player{
			Pid: player.Pid,
			P: &pb.Position{
				X: player.X,
				Y: player.Y,
				Z: player.Z,
				V: player.V,
			},
		}
		players_proto_msg = append(players_proto_msg, p)
	}

	//3.1.2 封装SyncPlayer protobuf 数据
	SyncPlayer_proto_msg := &pb.SyncPlayers{
		Ps: players_proto_msg[:],
	}

	//3.2 将组装好的数据发送给当前玩家客户端
	p.SendMsg(202, SyncPlayer_proto_msg)
}

// 广播并更新当前玩家坐标
func (p *Player) UpdatePos(x float32, y float32, z float32, v float32) {
	//触发消失视野和添加视野业务
	//计算旧格子gID
	oldGID := WorldMgrObj.AoiMgr.GetGidbyPos(p.X, p.Z)
	//计算新格子gID
	newGID := WorldMgrObj.AoiMgr.GetGidbyPos(x, z)

	//更新玩家的位置信息
	p.X = x
	p.Y = y
	p.Z = z
	p.V = v
	if oldGID != newGID {
		//触发gird切换
		//把pID从就的aoi格子中删除
		WorldMgrObj.AoiMgr.RemovePidfromGrid(int(p.Pid), oldGID)
		//把pID添加到新的aoi格子中去
		WorldMgrObj.AoiMgr.AddPidToGrid(int(p.Pid), newGID)

		_ = p.OnExchangeAoiGrID(oldGID, newGID)
	}

	//组装protobuf协议，发送位置给周围玩家
	proto_msg := &pb.BroadCast{
		Pid: p.Pid,
		Tp:  4, //4 移动之后的坐标信息
		Data: &pb.BroadCast_P{
			P: &pb.Position{
				X: p.X,
				Y: p.Y,
				Z: p.Z,
				V: p.V,
			},
		},
	}

	//获取当前玩家周边全部玩家AOI九宫格之内的玩家
	players := p.GetSurrundingPlayers()

	//依次给每个玩家对应的客户端发送当前玩家位置更新的消息
	//向周边的每个玩家发送MsgID:200消息，移动位置更新消息
	for _, player := range players {
		player.SendMsg(200, proto_msg)
	}

}
func (p *Player) GetSurrundingPlayers() []*Player {
	//得到当前AOI九宫格内的所有玩家PID
	pids := WorldMgrObj.AoiMgr.GetPidsbyPos(p.X, p.Z)
	//将所有pid对应的Player放到Player切片中
	players := make([]*Player, 0, len(pids))
	for _, pid := range pids {
		players = append(players, WorldMgrObj.GetPlayerByPid(int32(pid)))
	}

	return players

}

// 玩家下线
func (p *Player) Offline() {
	//1 获取周围AOI九宫格内的玩家
	players := p.GetSurrundingPlayers()

	//2 封装MsgID:201消息
	proto_msg := &pb.SyncPid{
		Pid: p.Pid,
	}

	//3 向周围玩家发送消息
	for _, player := range players {
		player.SendMsg(201, proto_msg)
	}

	//4 世界管理器将当前玩家从AOI中摘除
	WorldMgrObj.AoiMgr.RemoveFromGridByPos(int(p.Pid), p.X, p.Z) //从格子中删除
	WorldMgrObj.RemovePlayerByid(p.Pid)
}

func (p *Player) OnExchangeAoiGrID(oldGID int, newGID int) error {
	//获取就的九宫格成员
	oldGrIDs := WorldMgrObj.AoiMgr.GetSurroundGridsByGid(oldGID)

	//为旧的九宫格成员建立哈希表,用来快速查找
	oldGrIDsMap := make(map[int]bool, len(oldGrIDs))
	for _, grID := range oldGrIDs {
		oldGrIDsMap[grID.GID] = true
	}

	//获取新的九宫格成员
	newGrIDs := WorldMgrObj.AoiMgr.GetSurroundGridsByGid(newGID)
	//为新的九宫格成员建立哈希表,用来快速查找
	newGrIDsMap := make(map[int]bool, len(newGrIDs))
	for _, grID := range newGrIDs {
		newGrIDsMap[grID.GID] = true
	}

	//------ > 处理视野消失 <-------
	offlineMsg := &pb.SyncPid{
		Pid: p.Pid,
	}

	//找到在旧的九宫格中出现,但是在新的九宫格中没有出现的格子
	leavingGrIDs := make([]*Grid, 0)
	for _, grID := range oldGrIDs {
		if _, ok := newGrIDsMap[grID.GID]; !ok {
			leavingGrIDs = append(leavingGrIDs, grID)
		}
	}

	//获取需要消失的格子中的全部玩家
	for _, grID := range leavingGrIDs {
		players := WorldMgrObj.GetPlayersByGID(grID.GID)
		for _, player := range players {
			//让自己在其他玩家的客户端中消失
			player.SendMsg(201, offlineMsg)

			//将其他玩家信息 在自己的客户端中消失
			anotherOfflineMsg := &pb.SyncPid{
				Pid: player.Pid,
			}
			p.SendMsg(201, anotherOfflineMsg)
			time.Sleep(200 * time.Millisecond)
		}
	}

	//------ > 处理视野出现 <-------

	//找到在新的九宫格内出现,但是没有在就的九宫格内出现的格子
	enteringGrIDs := make([]*Grid, 0)
	for _, grID := range newGrIDs {
		if _, ok := oldGrIDsMap[grID.GID]; !ok {
			enteringGrIDs = append(enteringGrIDs, grID)
		}
	}

	onlineMsg := &pb.BroadCast{
		Pid: p.Pid,
		Tp:  2,
		Data: &pb.BroadCast_P{
			P: &pb.Position{
				X: p.X,
				Y: p.Y,
				Z: p.Z,
				V: p.V,
			},
		},
	}

	//获取需要显示格子的全部玩家
	for _, grID := range enteringGrIDs {
		players := WorldMgrObj.GetPlayersByGID(grID.GID)

		for _, player := range players {
			//让自己出现在其他人视野中
			player.SendMsg(200, onlineMsg)

			//让其他人出现在自己的视野中
			anotherOnlineMsg := &pb.BroadCast{
				Pid: player.Pid,
				Tp:  2,
				Data: &pb.BroadCast_P{
					P: &pb.Position{
						X: player.X,
						Y: player.Y,
						Z: player.Z,
						V: player.V,
					},
				},
			}

			time.Sleep(200 * time.Millisecond)
			p.SendMsg(200, anotherOnlineMsg)
		}
	}

	return nil

}
