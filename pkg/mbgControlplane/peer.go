package mbgControlplane

import (
	log "github.com/sirupsen/logrus"
	"github.ibm.com/mbg-agent/cmd/mbg/state"
	"github.ibm.com/mbg-agent/pkg/protocol"
)

func AddPeer(p protocol.PeerRequest) {
	//Update MBG state
	state.UpdateState()
	state.AddMbgNbr(p.Id, p.Ip, p.Cport)
}

func GetPeer(peerID string) protocol.PeerRequest {
	//Update MBG state
	state.UpdateState()
	MbgArr := state.GetMbgArr()
	m, ok := MbgArr[peerID]
	if ok {
		return protocol.PeerRequest{Id: m.Id, Ip: m.Ip, Cport: m.Cport.External}
	} else {
		log.Infof("MBG %s is not exist in the MBG peers list ", peerID)
		return protocol.PeerRequest{}
	}

}