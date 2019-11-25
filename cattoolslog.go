package main

import (
	"encoding/csv"
	"fmt"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
	"io/ioutil"
	"os"
	"os/user"
	"regexp"
	"time"
)

const (
	OK       = "OK"
	WAR      = "WARNING"
	CRI      = "CRITICAL"
	UNK      = "UNKNOWN"
	OK_CODE  = 0
	WAR_CODE = 1
	CRI_CODE = 2
	UNK_CODE = 3
)

type DeviceHistory struct {
	timestamp int64
	backup    string
	changes   string
}

type catToolsLogItem struct {
	timestamp  time.Time
	group      string
	deviceName string
	address    string
	deviceType string
	backup     string
	changes    string
	htmlReport string
	txtReport  string
}

type CatToolsLog struct {
	logs    []catToolsLogItem
	device  map[string][]DeviceHistory
	status  int
	message string
}

func max(x, y int) int {
	if x < y {
		return y
	} else {
		return x
	}
}

func (ctl *CatToolsLog) Load(host string, username string, password string, identity string, port int) {
	var signer ssh.Signer

	// Initialize properties
	ctl.message = ""
	ctl.status = UNK_CODE

	// replacing tilde char by real home directory
	home, _ := user.Current()
	re := regexp.MustCompile(`^~(.*)$`)
	identity = re.ReplaceAllString(identity, home.HomeDir+"${1}")

	key, err := ioutil.ReadFile(identity)
	if err == nil {
		// Create the Signer for this private key.
		signer, err = ssh.ParsePrivateKey(key)
		if err != nil {
			signer = nil
		}
	} else {
		signer = nil
	}

	var auths []ssh.AuthMethod
	if signer != nil {
		auths = append(auths, ssh.PublicKeys(signer))
	}
	if password != "" {
		auths = append(auths, ssh.Password(password))
	}

	config := &ssh.ClientConfig{
		User:            username,
		Auth:            auths,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	client, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", host, port), config)
	if err != nil {
		fmt.Printf("%s Error connecting SFTP server: %s\n", CRI, err)
		os.Exit(CRI_CODE)
	}
	defer client.Close()

	sftpClient, err := sftp.NewClient(client)
	if err != nil {
		fmt.Printf("%s Error connecting SFTP server: %s\n", CRI, err)
		os.Exit(CRI_CODE)
	}
	defer sftpClient.Close()

	f, err := sftpClient.Open("/Device.Backup.Running Config.txt")
	if err != nil {
		fmt.Printf("%s Error opening 'Device.Backup.Running Config.txt' file: %s\n", CRI, err)
		os.Exit(CRI_CODE)
	}
	defer f.Close()

	reader := csv.NewReader(f)
	reader.Comma = '\t'
	content, err := reader.ReadAll()

	var record catToolsLogItem
	// Reading only 100 last rows
	for _, item := range content[max(1, len(content)-100):] {
		record.timestamp, _ = time.ParseInLocation("2006/01/02 15:04:05", item[0], time.Local)
		record.group = item[1]
		record.deviceName = item[2]
		record.address = item[3]
		record.deviceType = item[4]
		record.backup = item[5]
		record.changes = item[6]
		record.htmlReport = item[7]
		record.txtReport = item[8]
		ctl.logs = append(ctl.logs, record)
	}

}

func (ctl *CatToolsLog) Analyze(interval int) {

	ctl.device = make(map[string][]DeviceHistory)
	for _, device := range ctl.logs {
		ctl.device[device.deviceName] = append(ctl.device[device.deviceName], DeviceHistory{device.timestamp.Unix(), device.backup, device.changes})
	}

	for key, value := range ctl.device {
		log := value[len(value)-1]
		lastCheck := log.timestamp
		now := time.Now().Unix()

		if now-lastCheck > 3600*24*int64(interval)+3600 {
			if ctl.status != CRI_CODE {
				ctl.status = CRI_CODE
			}
			if ctl.message == "" {
				ctl.message = fmt.Sprintf("%s not backuped for more than %d day(s)", key, interval)
			} else {
				ctl.message = fmt.Sprintf("%s / %s not backuped for more than %d day(s)", ctl.message, key, interval)
			}
			continue
		}
		if log.backup != "OK" {
			if ctl.status != CRI_CODE {
				ctl.status = CRI_CODE
			}
			if ctl.message == "" {
				ctl.message = fmt.Sprintf("%s backup error: %s", key, log.backup)
			} else {
				ctl.message = fmt.Sprintf("%s / %s backup error: %s", ctl.message, key, log.backup)
			}
			continue
		}
		if log.changes == "Changed!" {
			if ctl.message == "" {
				ctl.message = fmt.Sprintf("%s config was changed", key)
			} else {
				ctl.message = fmt.Sprintf("%s / %s config was changed", ctl.message, key)
			}
		} else if log.backup == "OK" {
			if ctl.status == UNK_CODE {
				ctl.status = OK_CODE
			}
			if ctl.message == "" {
				ctl.message = fmt.Sprintf("%s OK", key)
			} else {
				ctl.message = fmt.Sprintf("%s / %s OK", ctl.message, key)
			}
		}
	}
}

func (ctl *CatToolsLog) ReturnResult() {
	if ctl.status == OK_CODE && ctl.message == "" {
		ctl.message = "CatTools Backup are Ok!"
	}
	switch ctl.status {
	case OK_CODE:
		ctl.message = fmt.Sprintf("%s %s", OK, ctl.message)
	case WAR_CODE:
		ctl.message = fmt.Sprintf("%s %s", WAR, ctl.message)
	case CRI_CODE:
		ctl.message = fmt.Sprintf("%s %s", CRI, ctl.message)
	default:
		ctl.message = ""
		ctl.message = fmt.Sprintf("%s %s condition %s", UNK, UNK, ctl.message)
	}
	fmt.Println(ctl.message)
	os.Exit(ctl.status)
}
