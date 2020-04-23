package routetable

import (
	"encoding/json"

	cni "github.com/phdata/go-libcni"
)

/*
	* DefaultRequestedAddressSleepTime is the default amount of time (ms) we wait
		between tries to select a requested address

	* DefaultPropagationTimeout is the amount of time (ms) we wait for the
		possibility that another host has selected the same address

	* DefaultRouteProtocol is the default protocol number for our installed routes
*/
const (
	DefaultRequestedAddressSleepTime = 100
	DefaultPropagationTimeout        = 100
	DefaultRouteProtocol             = 192
)

// NewConfig returns a new vxlan config from the byte array
func NewConfig(confBytes []byte) (*cni.Config, error) {
	conf := &cni.Config{}
	err := json.Unmarshal(confBytes, conf)
	if err != nil {
		return nil, err
	}

	return conf, nil
}
