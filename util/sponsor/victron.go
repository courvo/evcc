package sponsor

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/evcc-io/evcc/api/proto/pb"
	"github.com/evcc-io/evcc/util/cloud"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// isVictron checks if the hardware is a victron device, only returns error if cloud check fails due to network issues
func isVictron() (bool, error) {
	fmt.Println("victron check")

	vd, err := victronDeviceInfo()

	// unable to retrieve all device info
	if err != nil || vd.ProductId == "" || vd.VrmId == "" || vd.Serial == "" || vd.Board == "" {
		fmt.Println("victron check failed", err, vd)
		return false, nil
	}

	conn, err := cloud.Connection()
	if err != nil {
		return false, err
	}

	client := pb.NewVictronClient(conn)
	if res, err := client.IsValidDevice(context.Background(), &pb.VictronRequest{
		ProductId: vd.ProductId,
		VrmId:     vd.VrmId,
		Serial:    vd.Serial,
		Board:     vd.Board,
	}); err == nil && res.Authorized {
		return true, nil
	} else {
		if s, ok := status.FromError(err); ok && s.Code() != codes.Unknown {
			return false, errors.New("unable to validate victron device")
		}
	}

	return false, nil
}

type victronDevice struct {
	ProductId string
	VrmId     string
	Serial    string
	Board     string
}

func commandExists(cmd string) error {
	_, err := exec.LookPath(cmd)
	return err
}

func executeCommand(ctx context.Context, cmd string, args ...string) (string, error) {
	command := exec.CommandContext(ctx, cmd, args...)
	output, err := command.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

func victronDeviceInfo() (victronDevice, error) {
	if runtime.GOOS != "linux" {
		return victronDevice{}, errors.New("non-linux os")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var vd victronDevice

	commands := []struct {
		field *string
		cmd   string
		args  []string
	}{
		{field: &vd.Board, cmd: "/usr/bin/board-compat"},
		{field: &vd.ProductId, cmd: "/usr/bin/product-id"},
		{field: &vd.VrmId, cmd: "/sbin/get-unique-id"},
		{field: &vd.Serial, cmd: "/opt/victronenergy/venus-eeprom/eeprom", args: []string{"--show", "serial-number"}},
	}

	for _, detail := range commands {
		if err := commandExists(detail.cmd); err != nil {
			return vd, errors.New("cmd not found: " + detail.cmd)
		}
		output, err := executeCommand(ctx, detail.cmd, detail.args...)
		if err != nil {
			return vd, err
		}
		*detail.field = output
	}

	return vd, nil
}