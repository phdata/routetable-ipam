package main

import (
	"fmt"
	"os"
	"strconv"

	cni "github.com/phdata/go-libcni"
	"github.com/phdata/routetable-ipam"
	log "github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"
)

func main() {
	var exitOutput []byte
	exitCode := 0
	lf, err := os.OpenFile("/var/log/route-table.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		exitCode, exitOutput = cni.PrepareExit(err, 99, "failed to open log file")
		return
	}
	defer lf.Close()
	log.SetOutput(lf)
	log.SetLevel(log.DebugLevel)

	defer func() {
		r := recover()
		if r != nil {
			err, ok := r.(error)
			if !ok {
				err = fmt.Errorf("panic: %v", r)
			}
			exitCode, exitOutput = cni.PrepareExit(err, 99, "panic during execution")
		}
		exit(exitCode, exitOutput)
	}()

	log.WithField("command", os.Getenv("CNI_COMMAND")).Debug()
	varNames := []string{"CNI_COMMAND", "CNI_CONTAINERID", "CNI_NETNS", "CNI_IFNAME", "CNI_ARGS", "CNI_PATH"}
	varMap := log.Fields{}
	for _, vn := range varNames {
		varMap[vn] = os.Getenv(vn)
	}
	log.WithFields(varMap).Debug("vars")

	//Read CNI standard environment variables
	vars := cni.NewVars()

	if vars.Command == "VERSION" {
		//report supported cni versions
		exitOutput = []byte(fmt.Sprintf("{\"cniVersion\": \"%v\", \"supportedVersions\": [\"%v\"]}", cni.CNIVersion, cni.CNIVersion))
		return
	}

	cidr, ok := vars.GetArg("CIDR")
	if !ok {
		exitCode, exitOutput = cni.PrepareExit(fmt.Errorf("CNI_ARGS must contain CIDR=<cidr>, where <cidr> represents the address/subnet from which to choose an address"), 7, "missing cidr arg")
		return
	}

	xf := 0
	sxf, ok := vars.GetArg("EXCLUDE_FIRST")
	if ok {
		xf, _ = strconv.Atoi(sxf)
	}

	xl := 0
	sxl, ok := vars.GetArg("EXCLUDE_LAST")
	if ok {
		xl, _ = strconv.Atoi(sxl)
	}

	switch vars.Command {
	case "ADD":
		addr, err := routetable.SelectAddress(cidr, xf, xl)
		if err != nil {
			exitCode, exitOutput = cni.PrepareExit(err, 11, "failed while attempting to select an address and install the route")
			return
		}
		ipVer := "4"
		if addr.IP.To4() == nil {
			ipVer = "6"
		}
		ips := make([]*cni.IP, 1)
		ips[0] = &cni.IP{
			Version: ipVer,
			Address: addr.String(),
		}
		result := &cni.Result{
			CNIVersion: cni.CNIVersion,
			IPs:        ips,
		}
		os.Stdout.Write(result.Marshal())
	case "DEL":
		//remove /32 route
		addr, _ := netlink.ParseIPNet(cidr)
		_, addrOnly := routetable.GetIPNets(addr.IP, addr)
		linkIndex, err := routetable.LinkIndexFromIPNet(addrOnly)
		if err != nil {
			log.WithError(err).Errorf("failed to get link index from address")
			return
		}
		routetable.DelRoute(linkIndex, addrOnly)
		return
	case "CHECK":
		return
		//if all "ADD" steps are correct
		//exit 0
		//else
		//exit error
	default:
		exitCode, exitOutput = cni.PrepareExit(fmt.Errorf("CNI_COMMAND was not set, or set to an invalid value"), 4, "invalid CNI_COMMAND")
		return
	}
}

func exit(code int, output []byte) {
	os.Stdout.Write(output)
	os.Exit(code)
}
