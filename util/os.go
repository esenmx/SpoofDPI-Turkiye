package util

import (
	"errors"
	"fmt"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
)

const (
	getDefaultNetworkCMD = "networksetup -listnetworkserviceorder | grep" +
		" `(route -n get default | grep 'interface' || route -n get -inet6 default | grep 'interface') | cut -d ':' -f2`" +
		" -B 1 | head -n 1 | cut -d ' ' -f 2-"
	darwinOS                     = "darwin"
	permissionErrorHelpTextMacOS = "Varsayılan olarak SpoofDPI kendisini sistem genelinde bir proxy sunucusu olarak ayarlamaya çalışır.\n" +
		"Bu işlem kök (root) erişimine ihtiyaç duyabilir:\n" +
		"'Settings > Privacy & Security > Advanced > Require\n" +
		" Bir yetkili şifresi girmeniz istenecektir.\n" +
		"SpoofDPI'ın sistem genelinde bir proxy olarak hareket etmesini istemiyorsanız" +
		" -system-proxy=false ayarlayınız."
)

func SetOsProxy(port uint16) error {
	if runtime.GOOS != darwinOS {
		return nil
	}

	network, err := getDefaultNetwork()
	if err != nil {
		return err
	}

	return setProxy(getProxyTypes(), network, "127.0.0.1", port)
}

func UnsetOsProxy() error {
	if runtime.GOOS != darwinOS {
		return nil
	}

	network, err := getDefaultNetwork()
	if err != nil {
		return err
	}

	return unsetProxy(getProxyTypes(), network)
}

func getDefaultNetwork() (string, error) {
	network, err := exec.Command("sh", "-c", getDefaultNetworkCMD).Output()
	if err != nil {
		return "", err
	} else if len(network) == 0 {
		return "", errors.New("no available networks")
	}
	return strings.TrimSpace(string(network)), nil
}

func getProxyTypes() []string {
	return []string{"webproxy", "securewebproxy"}
}

func setProxy(proxyTypes []string, network, domain string, port uint16) error {
	portStr := strconv.FormatUint(uint64(port), 10)
	for _, proxyType := range proxyTypes {
		args := []string{"-set" + proxyType, network, domain, portStr}
		if err := networkSetup(args); err != nil {
			return fmt.Errorf("setting %s: %w", proxyType, err)
		}
	}
	return nil
}

func unsetProxy(proxyTypes []string, network string) error {
	for _, proxyType := range proxyTypes {
		args := []string{"-set" + proxyType + "state", network, "off"}
		if err := networkSetup(args); err != nil {
			return fmt.Errorf("unsetting %s: %w", proxyType, err)
		}
	}
	return nil
}

func networkSetup(args []string) error {
	cmd := exec.Command("networksetup", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		msg := string(out)
		if isMacOSPermissionError(err) {
			msg += permissionErrorHelpTextMacOS
		}
		return fmt.Errorf("%s: %s", cmd.String(), msg)
	}
	return nil
}

func isMacOSPermissionError(err error) bool {
	if runtime.GOOS != darwinOS {
		return false
	}

	var exitErr *exec.ExitError
	ok := errors.As(err, &exitErr)
	return ok && exitErr.ExitCode() == 14
}
