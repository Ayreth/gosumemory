package main

import (
	"bufio"
	"encoding/binary"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/Andoryuuta/kiwi"
	"github.com/gorilla/websocket"
	"github.com/inconshreveable/go-update"
	"github.com/spf13/cast"
)

var proc, procerr = kiwi.GetProcessByFileName("osu!.exe")

var osuBase uintptr
var bpmBase uintptr
var leaderBase uintptr
var songsFolderAddr uintptr
var osuStatus uint16
var currentBeatmapData uintptr
var playContainer uintptr
var inMenuAppliedModsBase uintptr
var playContainerBase uintptr
var serverBeatmapString string
var outStrLoop string
var baseDir string
var playTimeBase uintptr
var playTime uintptr
var currentBeatmapDataBase uint32
var currentBeatmapDataFirtLevel uint32
var playContainerBaseAddr uint32
var playContainerFirstlevel uint32
var playContainer38 uint32
var fullPathToOsu string
var osuFileStdIN string
var currentHitObjectStats string = ""

//Gameplay pp related
var ourTime []int
var lastObjectInt int
var lastObject string
var ppAcc string
var ppCombo string
var pp100 string
var pp50 string
var ppMiss string
var ppMods string = ""
var pp string = ""
var leaderStruct uint32

//Menu pp related
var ppSS string = ""
var pp99 string = ""
var pp98 string = ""
var pp97 string = ""
var pp96 string = ""
var pp95 string = ""
var ppifFC string = ""
var innerBGPath string = ""
var updateTime int
var isRunning = 0
var isInit = 0
var workingDirectory string
var operatingSystem int8
var uintptrOsuStatus uintptr
var jsonByte []byte
var reqRestart int8 = 0
var tempCounter int16 = 0

var (
	osuRegex = regexp.MustCompile(`.*osu!\.exe.*`)
)

func Cmd(cmd string, shell bool) []byte {

	if shell {
		out, err := exec.Command("sh", "-c", cmd).Output()
		if err != nil {
			//	println("some error found", err)
		}
		return out
	} else {
		out, err := exec.Command(cmd).Output()
		if err != nil {
			//	println("some error found2", err)
		}
		return out

	}
}

func OsuStatusAddr() uintptr {
	proc, procerr = kiwi.GetProcessByFileName("osu!.exe")
	for procerr != nil {
		fmt.Println("It looks like we got a client restart mid getting offsets, trying to recover.. (waiting for the game to launch)")
		proc, procerr = kiwi.GetProcessByFileName("osu!.exe")
		time.Sleep(1 * time.Second)

	}
	if operatingSystem == 1 {
		cmd, err := exec.Command("deps/OsuStatusAddr.exe").Output()
		if err != nil {
			OsuStatusAddr()
		}
		outStr := cast.ToString(cmd)
		outStr = strings.Replace(outStr, "\n", "", -1)
		outStr = strings.Replace(outStr, "\r", "", -1)
		outInt := cast.ToUint32(outStr)

		osuBase = uintptr(outInt)
		fmt.Printf("OsuStatusAddr: 0x%x\n", osuBase)

	} else {
		maps, err := readMaps(int(proc.PID))
		if err != nil {
			//log.Fatal(err)
		}
		mem, err := os.Open(fmt.Sprintf("/proc/%d/mem", proc.PID)) //TODO: Should only read the mem once
		if err != nil {
			//log.Fatal(err)
		}
		defer mem.Close()
		base, err := scan(mem, maps, "48 83 F8 04 73 1E")
		if err != nil {
			//log.Fatal(err)
		}
		fmt.Printf("OsuStatusAddr: 0x%x\n", base)
		// osuBaseString := "0x" + yosuBase
		osuBase = uintptr(base)
	}

	for osuBase == 0 {
		fmt.Println("osu! is not fully loaded yet, waiting...")
		time.Sleep(1 * time.Second)
		OsuStatusAddr()
	}

	//println(CurrentBeatmapFolderString())
	return osuBase
}
func OsuSongsFolderAddr() uintptr {
	proc, procerr = kiwi.GetProcessByFileName("osu!.exe")
	for procerr != nil {
		fmt.Println("It looks like we got a client restart mid getting offsets, trying to recover.. (waiting for the game to launch)")
		proc, procerr = kiwi.GetProcessByFileName("osu!.exe")
		time.Sleep(1 * time.Second)

	}
	if operatingSystem == 1 {
		cmd, err := exec.Command("deps/OsuSongsFolderAddr.exe").Output()
		if err != nil {
			log.Println("Could not get Songs folder from memory, please manually specify it (see --help)")
		}
		outStr := cast.ToString(cmd)
		outStr = strings.Replace(outStr, "\n", "", -1)
		outStr = strings.Replace(outStr, "\r", "", -1)
		outInt := cast.ToUint32(outStr)

		osuBase = uintptr(outInt)
		fmt.Printf("OsuSongsFolderAddr: 0x%x\n", osuBase)

	} else {
		if workingDirectory == "auto" {
			log.Fatalln("We don't support automatic search for Songs folder path, please start the program with --help")
		}
	}

	if osuBase == 0 {
		log.Println("Could not find OsuSongsFolderAddr, is osu! running? (retrying)")
		restart()
	}

	//println(CurrentBeatmapFolderString())
	return osuBase
}

func OsuBPMAddr() uintptr {
	if operatingSystem == 1 {
		cmd, err := exec.Command("deps/OsuBPMAddr.exe").Output()
		if err != nil {
			//fmt.Println(err)
		}
		outStr := cast.ToString(cmd)
		outStr = strings.Replace(outStr, "\n", "", -1)
		outStr = strings.Replace(outStr, "\r", "", -1)
		outInt := cast.ToUint32(outStr)

		osuBase = uintptr(outInt)
		fmt.Printf("OsuBPMAddr: 0x%x\n", osuBase)

	} else {
		maps, err := readMaps(int(proc.PID))
		if err != nil {
			fmt.Println("It looks like we got a client restart mid getting offsets, trying to recover.. (waiting for the game to launch)")
			restart()
		}
		mem, err := os.Open(fmt.Sprintf("/proc/%d/mem", proc.PID)) //TODO: Should only read the mem once
		if err != nil {
			fmt.Println("It looks like we got a client restart mid getting offsets, trying to recover.. (waiting for the game to launch)")
			restart()
		}
		defer mem.Close()
		base, err := scan(mem, maps, "?? ?? ?? ?? 8B 40 08 89 86 4C 01 00 00 C6")
		if err != nil {
			fmt.Println("It looks like we got a client restart mid getting offsets, trying to recover.. (waiting for the game to launch)")
			restart()
		}
		fmt.Printf("OsuBPMAddr: 0x%x\n", base)
		// osuBaseString := "0x" + yosuBase
		osuBase = uintptr(base)
	}

	if osuBase == 0 {
		log.Println("Could not find OsuBaseAddr, is osu! running? (retrying)")
		restart()
	}

	//println(CurrentBeatmapFolderString())
	return osuBase
}

func OsuLeaderAddr() uintptr {
	if operatingSystem == 1 {
		cmd, err := exec.Command("deps/OsuLeaderAddr.exe").Output()
		if err != nil {
			//fmt.Println(err)
		}
		outStr := cast.ToString(cmd)
		outStr = strings.Replace(outStr, "\n", "", -1)
		outStr = strings.Replace(outStr, "\r", "", -1)
		outInt := cast.ToUint32(outStr)

		osuBase = uintptr(outInt)
		fmt.Printf("OsuLeaderAddr: 0x%x\n", osuBase)

	} else {
		maps, err := readMaps(int(proc.PID))
		if err != nil {
			fmt.Println("It looks like we got a client restart mid getting offsets, trying to recover.. (waiting for the game to launch)")
			restart()
		}
		mem, err := os.Open(fmt.Sprintf("/proc/%d/mem", proc.PID)) //TODO: Should only read the mem once
		if err != nil {
			fmt.Println("It looks like we got a client restart mid getting offsets, trying to recover.. (waiting for the game to launch)")
			restart()
		}
		defer mem.Close()
		base, err := scan(mem, maps, "A1 ?? ?? ?? ?? 8B 50 04 8B 0D")
		if err != nil {
			fmt.Println("It looks like we got a client restart mid getting offsets, trying to recover.. (waiting for the game to launch)")
			restart()
		}
		fmt.Printf("OsuBPMAddr: 0x%x\n", base)
		// osuBaseString := "0x" + yosuBase
		osuBase = uintptr(base)
	}

	if osuBase == 0 {
		log.Println("Could not find OsuBaseAddr, is osu! running? (retrying)")
		restart()
	}

	//println(CurrentBeatmapFolderString())
	return osuBase
}

func OsuBaseAddr() uintptr {
	if operatingSystem == 1 {
		cmd, err := exec.Command("deps/OsuBaseAddr.exe").Output()
		if err != nil {
			//fmt.Println(err)
		}
		outStr := cast.ToString(cmd)
		outStr = strings.Replace(outStr, "\n", "", -1)
		outStr = strings.Replace(outStr, "\r", "", -1)
		outInt := cast.ToUint32(outStr)
		osuBase = uintptr(outInt)
		fmt.Printf("OsuBaseAddr: 0x%x\n", osuBase)
	} else {
		maps, err := readMaps(int(proc.PID))
		if err != nil {
			fmt.Println("It looks like we got a client restart mid getting offsets, trying to recover.. (waiting for the game to launch)")
			restart()
		}
		mem, err := os.Open(fmt.Sprintf("/proc/%d/mem", proc.PID)) //TODO: Should only read the mem once
		if err != nil {
			fmt.Println("It looks like we got a client restart mid getting offsets, trying to recover.. (waiting for the game to launch)")
			restart()
		}
		defer mem.Close()
		base, err := scan(mem, maps, "F8 01 74 04 83 65")
		if err != nil {
			fmt.Println("It looks like we got a client restart mid getting offsets, trying to recover.. (waiting for the game to launch)")
			restart()
		}
		fmt.Printf("OsuBaseAddr: 0x%x\n", base)
		// osuBaseString := "0x" + yosuBase
		osuBase = uintptr(base)
	}

	if osuBase == 0 {
		log.Println("Could not find OsuBaseAddr, is osu! running? (retrying)")
		restart()
	}

	//println(CurrentBeatmapFolderString())
	return osuBase
}

func OsuInMenuModsAddr() uintptr {
	if operatingSystem == 1 {
		cmd, err := exec.Command("deps/InMenuAppliedModsAddr.exe").Output()
		if err != nil {
			//fmt.Println(err)
		}
		outStr := cast.ToString(cmd)
		outStr = strings.Replace(outStr, "\n", "", -1)
		outStr = strings.Replace(outStr, "\r", "", -1)
		outInt := cast.ToUint32(outStr)

		osuBase = uintptr(outInt)
		fmt.Printf("OsuInMenuModsAddr: 0x%x\n", osuBase)
	} else {
		maps, err := readMaps(int(proc.PID))
		if err != nil {
			fmt.Println("It looks like we got a client restart mid getting offsets, trying to recover.. (waiting for the game to launch)")
			restart()
		}
		mem, err := os.Open(fmt.Sprintf("/proc/%d/mem", proc.PID)) //TODO: Should only read the mem once
		if err != nil {
			fmt.Println("It looks like we got a client restart mid getting offsets, trying to recover.. (waiting for the game to launch)")
			restart()
		}
		defer mem.Close()
		base, err := scan(mem, maps, "C8 FF ?? ?? ?? ?? ?? 81 0D ?? ?? ?? ?? 00 08 00 00")
		if err != nil {
			fmt.Println("It looks like we got a client restart mid getting offsets, trying to recover.. (waiting for the game to launch)")
			restart()
		}
		fmt.Printf("OsuInMenuModsAddr: 0x%x\n", base)
		// osuBaseString := "0x" + yosuBase
		osuBase = uintptr(base)
	}

	if osuBase == 0 {
		log.Println("Could not find OsuBaseAddr, is osu! running? (retrying)")
		restart()
	}

	//println(CurrentBeatmapFolderString())
	return osuBase
}

func OsuPlayTimeAddr() uintptr {
	if operatingSystem == 1 {
		cmd, err := exec.Command("deps/OsuPlayTimeAddr.exe").Output()
		if err != nil {
			//fmt.Println(err)
		}
		outStr := cast.ToString(cmd)
		outStr = strings.Replace(outStr, "\n", "", -1)
		outStr = strings.Replace(outStr, "\r", "", -1)
		outInt := cast.ToUint32(outStr)

		osuBase = uintptr(outInt)
		fmt.Printf("OsuPlayTimeAddr: 0x%x\n", osuBase)
	} else {
		maps, err := readMaps(int(proc.PID))
		if err != nil {
			fmt.Println("It looks like we got a client restart mid getting offsets, trying to recover.. (waiting for the game to launch)")
			restart()
		}
		mem, err := os.Open(fmt.Sprintf("/proc/%d/mem", proc.PID)) //TODO: Should only read the mem once
		if err != nil {
			fmt.Println("It looks like we got a client restart mid getting offsets, trying to recover.. (waiting for the game to launch)")
			restart()
		}
		defer mem.Close()
		base, err := scan(mem, maps, "5E 5F 5D C3 A1 ?? ?? ?? ?? 89 ?? 04")
		if err != nil {
			fmt.Println("It looks like we got a client restart mid getting offsets, trying to recover.. (waiting for the game to launch)")
			restart()
		}
		fmt.Printf("OsuPlayTimeAddr: 0x%x\n", base)
		// osuBaseString := "0x" + yosuBase
		osuBase = uintptr(base)
	}

	if osuBase == 0 {
		log.Println("Could not find OsuBaseAddr, is osu! running? (retrying)")
		restart()
	}

	//println(CurrentBeatmapFolderString())
	return osuBase
}

func OsuplayContainer() uintptr {
	if operatingSystem == 1 {

		cmd, err := exec.Command("deps/OsuPlayContainer.exe").Output()
		if err != nil {
			//fmt.Println(err)
		}
		outStr := cast.ToString(cmd)
		outStr = strings.Replace(outStr, "\n", "", -1)
		outStr = strings.Replace(outStr, "\r", "", -1)
		outInt := cast.ToUint32(outStr)

		osuBase = uintptr(outInt)
		fmt.Printf("OsuPlayContainer: 0x%x\n", osuBase)
	} else {
		maps, err := readMaps(int(proc.PID))
		if err != nil {
			fmt.Println("It looks like we got a client restart mid getting offsets, trying to recover.. (waiting for the game to launch)")
			restart()
		}
		mem, err := os.Open(fmt.Sprintf("/proc/%d/mem", proc.PID)) //TODO: Should only read the mem once
		if err != nil {
			fmt.Println("It looks like we got a client restart mid getting offsets, trying to recover.. (waiting for the game to launch)")
			restart()
		}
		defer mem.Close()
		base, err := scan(mem, maps, "85 C9 74 1F 8D 55 F0 8B 01")
		if err != nil {
			fmt.Println("It looks like we got a client restart mid getting offsets, trying to recover.. (waiting for the game to launch)")
			restart()
		}
		fmt.Printf("base: 0x%x\n", base)
		// osuBaseString := "0x" + yosuBase
		osuBase = uintptr(base)
	}

	if osuBase == 0 {
		log.Println("Could not find OsuBaseAddr, is osu! running? (retrying)")
		restart()
	}

	//println(CurrentBeatmapFolderString())
	return osuBase
}
func InitBaseStuff() {
	songsFolderAddr = OsuSongsFolderAddr()
	osuBase = OsuBaseAddr()
	currentBeatmapData = (osuBase - 0xC)
	playTimeBase = OsuPlayTimeAddr()
	playContainer = OsuplayContainer()
	playContainerBase = (playContainer - 0x4)
	playTime = (playTimeBase + 0x5)
	inMenuAppliedModsBase = (OsuInMenuModsAddr() + 0x9)
	bpmBase = OsuBPMAddr()
	leaderBase = (OsuLeaderAddr() + 0x1)
}

func LeaderPosition() uint32 {
	baseLevel, err := proc.ReadUint32(uintptr(ResolveLeaderBoardStruct() + 0x8))
	if err != nil {
		//log.Println("CurrentBeatmapSetID result pointer failure")
		return 0x0
	}
	firstLevel, err := proc.ReadUint32(uintptr(baseLevel + 0x24))
	if err != nil {
		//log.Println("CurrentBeatmapSetID result pointer failure")
		return 0x0
	}
	secondLevel, err := proc.ReadUint32(uintptr(firstLevel + 0x10))
	if err != nil {
		//log.Println("CurrentBeatmapSetID result pointer failure")
		return 0x0
	}
	result, err := proc.ReadUint32(uintptr(secondLevel + 0x2C))
	if err != nil {
		//log.Println("CurrentBeatmapSetID result pointer failure")
		return 0x0
	}
	fmt.Printf("The chain is: %x %x %x %x\n", baseLevel, firstLevel, secondLevel, cast.ToUint32(result))
	return result
}
func ResolveLeaderBoardStruct() uint32 {
	baseLevel, err := proc.ReadUint32(uintptr(leaderBase))
	if err != nil {
		//log.Println("CurrentBeatmapSetID result pointer failure")
		return 0x0
	}
	firstLevel, err := proc.ReadUint32(uintptr(baseLevel + 0x4))
	if err != nil {
		//log.Println("CurrentBeatmapSetID result pointer failure")
		return 0x0
	}
	secondLevel, err := proc.ReadUint32(uintptr(firstLevel + 0x74))
	if err != nil {
		//log.Println("CurrentBeatmapSetID result pointer failure")
		return 0x0
	}
	thirdLevel, err := proc.ReadUint32(uintptr(secondLevel + 0x24))
	if err != nil {
		//log.Println("CurrentBeatmapSetID result pointer failure")
		return 0x0
	}
	fourthLevel, err := proc.ReadUint32(uintptr(thirdLevel + 0x4))
	if err != nil {
		//log.Println("CurrentBeatmapSetID result pointer failure")
		return 0x0
	}
	result, err := proc.ReadUint32(uintptr(fourthLevel + 0x4))
	if err != nil {
		//log.Println("CurrentBeatmapSetID result pointer failure")
		return 0x0
	}
	return result
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func reader(conn *websocket.Conn) {
	for {
		messageType, p, err := conn.ReadMessage()
		if err != nil {
			log.Println(err)
			return
		}
		fmt.Println(string(p))

		if err := conn.WriteMessage(messageType, p); err != nil {
			log.Println(err)
			return
		}

	}
}

func wsEndpoint(w http.ResponseWriter, r *http.Request) {
	upgrader.CheckOrigin = func(r *http.Request) bool { return true }

	// upgrade this connection to a WebSocket
	// connection
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
	}
	proc, procerr = kiwi.GetProcessByFileName("osu!.exe")
	for procerr != nil { //TODO: refactor
		ws.WriteMessage(1, []byte("osu!.exe not found"))
		if operatingSystem == 1 {
			log.Println("is osu! running? (osu! process was not found, waiting...)")
			time.Sleep(1 * time.Second)
			proc, procerr = kiwi.GetProcessByFileName("osu!.exe")
		} else {
			log.Println("is osu! running? (osu! process was not found, waiting...)")
			time.Sleep(1 * time.Second)
			proc, procerr = kiwi.GetProcessByFileName("osu!.exe")
		}
	}
	if isRunning == 0 {
		fmt.Println("Client Connected, please go to the SongSelect and check this console back.")
		//time.Sleep(7 * time.Second)            //hack to wait for the game
		StaticOsuStatusAddr := OsuStatusAddr() //we should only check for this address once.
		osuStatusOffset, err := proc.ReadUint32(StaticOsuStatusAddr - 0x4)
		for err != nil {
			log.Println("osu!status offset was not found, retrying...")
			StaticOsuStatusAddr = OsuStatusAddr()
			osuStatusOffset, err = proc.ReadUint32(StaticOsuStatusAddr - 0x4)
			time.Sleep(250 * time.Millisecond)
		}
		uintptrOsuStatus = uintptr(osuStatusOffset)
		osuStatusValue, err := proc.ReadUint16(uintptrOsuStatus)
		if err != nil {
			ws.WriteMessage(1, []byte("osu!status value was not found"))
			log.Fatalln("osu!status value was not found, are you sure that osu!stable is running? If so, please report this to GitHub!")
		}
		for {
			fmt.Println("please go to songselect or playmode in order to proceed!")
			osuStatusValue, err = proc.ReadUint16(uintptrOsuStatus)
			if err != nil {
				fmt.Println("It looks like we lost the process, performing a restart...")
				restart()
			}
			if osuStatusValue == 2 {
				break
			}
			if osuStatusValue == 5 {
				break
			}
			time.Sleep(500 * time.Millisecond)

			// time.Sleep(1 * time.Second)

		}

		//time.Sleep(5 * time.Second)

		if isInit == 0 { //TODO: This should be global, possible CPU leakage
			isInit = 1
			InitBaseStuff()
			if workingDirectory == "auto" && operatingSystem == 1 {
				workingDirectory = ResolveSongsFolder()
				baseDir = workingDirectory
				fmt.Println("Found Songs directory in: " + baseDir)
				http.Handle("/Songs/", http.StripPrefix("/Songs/", http.FileServer(http.Dir(workingDirectory))))
			} else {
				baseDir = workingDirectory
				http.Handle("/Songs/", http.StripPrefix("/Songs/", http.FileServer(http.Dir(workingDirectory))))
			}
		}

		if CurrentPlayTime() == -1 {
			fmt.Println("Failed to get the correct offsets, retrying...")
			restart()

		}

	}

	fmt.Println("it seems that we got the correct offsets, you are good to go!")
	log.Println("Client Connected")

	var tempCurrentBeatmapOsu string
	// var tempCurrentAppliedMods int32

	if isRunning == 0 { //TODO: Is this even correct?
		isRunning = 1
		for {
			osuStatusValue, err := proc.ReadUint16(uintptrOsuStatus)
			osuStatus = osuStatusValue
			var proc, procerr = kiwi.GetProcessByFileName("osu!.exe")
			for procerr != nil {
				reqRestart = 1
				fmt.Println("reqRestart = 1")
				log.Println("is osu! running? (osu! process was not found, waiting...)")
				proc, procerr = kiwi.GetProcessByFileName("osu!.exe")
				time.Sleep(1 * time.Second)
			}
			if reqRestart == 1 {
				fmt.Println("It looks like we have a client restart!")
				reqRestart = 0
				fmt.Println("reqRestart = 0")
				restart()

			}

			//init base stuff
			currentBeatmapDataBase, err = proc.ReadUint32(currentBeatmapData)
			if err != nil {
				log.Println("currentBeatmapDataBase Base level failure")
			}
			currentBeatmapDataFirtLevel, err = proc.ReadUint32(uintptr(currentBeatmapDataBase))
			if err != nil {
				log.Println("currentBeatmapDataFirtLevel First level pointer failure")
			}
			if osuStatus == 2 {
				playContainerBaseAddr, err = proc.ReadUint32(playContainerBase)
				if err != nil {
					log.Println("playContainerBaseAddr Base level failure")
				}
				playContainerFirstlevel, err = proc.ReadUint32(uintptr(playContainerBaseAddr))
				if err != nil {
					log.Println("playContainerFirstlevel pointer failure")
				}
				playContainer38, err = proc.ReadUint32(uintptr(playContainerFirstlevel) + 0x38)
				if err != nil {
					//	log.Println("playContainer38 pointer failure")

				}
			}

			type PlayContainer struct {
				CurrentHit300c  int16   `json:"300"`
				CurrentHit100c  int16   `json:"100"`
				CurrentHit50c   int16   `json:"50"`
				CurrentHitMiss  int16   `json:"miss"`
				CurrentAccuracy float64 `json:"accuracy"`
				CurrentScore    int32   `json:"score"`
				CurrentCombo    int32   `json:"combo"`
				CurrentGameMode int32   `json:"gameMode"`
				CurrentMaxCombo int32   `json:"maxCombo"`
				// CurrentPlayerHP         int8    `json:"playerHP"`
				// CurrentPlayerHPSmoothed int8    `json:"playerHPSmoothed"`
				Pp     string `json:"pp"`
				PPifFC string `json:"ppIfFC"`
			}
			type EverythingInMenu struct {
				CurrentState                uint16  `json:"osuState"`
				CurrentBeatmapID            uint32  `json:"bmID"`
				CurrentBeatmapSetID         uint32  `json:"bmSetID"`
				CurrentBeatmapCS            float32 `json:"CS"`
				CurrentBeatmapAR            float32 `json:"AR"`
				CurrentBeatmapOD            float32 `json:"OD"`
				CurrentBeatmapHP            float32 `json:"HP"`
				CurrentBeatmapString        string  `json:"bmInfo"`
				CurrentBeatmapFolderString  string  `json:"bmFolder"`
				CurrentBeatmapOsuFileString string  `json:"pathToBM"`
				CurrentHitObjectStats       string  `json:"bmStats"`
				CurrentPlayTime             int32   `json:"bmCurrentTime"`
				InnerBGPath                 string  `json:"innerBG"`
				CurrentAppliedMods          int32   `json:"appliedMods"`
				PpMods                      string  `json:"appliedModsString"`
				PpSS                        string  `json:"ppSS"`
				Pp99                        string  `json:"pp99"`
				Pp98                        string  `json:"pp98"`
				Pp97                        string  `json:"pp97"`
				Pp96                        string  `json:"pp96"`
				Pp95                        string  `json:"pp95"`
			}

			type EverythingInMenu2 struct { //order sets here
				D EverythingInMenu `json:"menuContainer"`
				P PlayContainer    `json:"gameplayContainer"`
			}

			PlayContainerStruct := PlayContainer{
				CurrentHit300c:  CurrentHit300c(),
				CurrentHit100c:  CurrentHit100c(),
				CurrentHit50c:   CurrentHit50c(),
				CurrentHitMiss:  CurrentHitMiss(),
				CurrentScore:    CurrentScore(),
				CurrentAccuracy: CurrentAccuracy(),
				CurrentCombo:    CurrentCombo(),
				CurrentGameMode: CurrentGameMode(),
				CurrentMaxCombo: CurrentMaxCombo(),
				// CurrentPlayerHP:         CurrentPlayerHP(),
				// CurrentPlayerHPSmoothed: CurrentPlayerHPSmoothed(),
				Pp:     pp,
				PPifFC: ppifFC,
			}

			//println(ValidCurrentBeatmapFolderString())
			// if strings.HasSuffix(CurrentBeatmapOsuFileString(), ".osu") == false {
			// 	println(".osu ends with ???")
			// }
			// if strings.HasSuffix(CurrentBeatmapString(), "]") == false {
			// 	println("beatmapstring ends with ???")
			// }
			MenuContainerStruct := EverythingInMenu{
				CurrentState:                osuStatus,
				CurrentBeatmapID:            CurrentBeatmapID(),
				CurrentBeatmapSetID:         CurrentBeatmapSetID(),
				CurrentBeatmapString:        CurrentBeatmapString(),
				CurrentBeatmapFolderString:  CurrentBeatmapFolderString(),
				CurrentBeatmapOsuFileString: CurrentBeatmapOsuFileString(),
				CurrentAppliedMods:          CurrentAppliedMods(),
				CurrentBeatmapAR:            CurrentBeatmapAR(),
				CurrentBeatmapOD:            CurrentBeatmapOD(),
				CurrentBeatmapCS:            CurrentBeatmapCS(),
				CurrentBeatmapHP:            CurrentBeatmapHP(),
				CurrentPlayTime:             CurrentPlayTime(),
				CurrentHitObjectStats:       currentHitObjectStats,
				InnerBGPath:                 innerBGPath,
				PpSS:                        ppSS,
				Pp99:                        pp99,
				Pp98:                        pp98,
				Pp97:                        pp97,
				Pp96:                        pp96,
				Pp95:                        pp95,
				PpMods:                      ppMods,
			}
			if osuStatus == 2 {
				leaderStruct = ResolveLeaderBoardStruct()
				if leaderStruct != 0x0 {
					fmt.Printf("LeaderboardtStructAddr is: 0x%x\n", uintptr(leaderStruct))
					fmt.Printf("LeaderboardPosition is: %x\n", LeaderPosition())

				}

				ppMods = ModsResolver(cast.ToUint32(MenuContainerStruct.CurrentAppliedMods)) //TODO: Refactor
			}

			if PlayContainerStruct.CurrentMaxCombo >= 1 {

				for _, hitObjectTime := range ourTime {

					if int32(hitObjectTime) >= MenuContainerStruct.CurrentPlayTime { //TODO: Fix inaccuracy

						lastObjectInt = SliceIndex(len(ourTime), func(i int) bool { return ourTime[i] == hitObjectTime })
						lastObject = cast.ToString(lastObjectInt)
						ppAcc = cast.ToString(PlayContainerStruct.CurrentAccuracy)
						pp100 = cast.ToString(PlayContainerStruct.CurrentHit100c)
						pp50 = cast.ToString(PlayContainerStruct.CurrentHit50c)
						ppCombo = cast.ToString(PlayContainerStruct.CurrentMaxCombo)
						ppMiss = cast.ToString(PlayContainerStruct.CurrentHitMiss)
						//TODO: Should only be called once
						pp = PP() //current pp
						ppifFC = PPifFC()

						break // Is the break really needed here?

					}
				}
			}
			if MenuContainerStruct.CurrentBeatmapOsuFileString != tempCurrentBeatmapOsu {
				ourTime = nil
				pp = ""
				ppifFC = ""

				tempCurrentBeatmapOsu = MenuContainerStruct.CurrentBeatmapOsuFileString
				fullPathToOsu = fmt.Sprintf(baseDir + "/" + MenuContainerStruct.CurrentBeatmapFolderString + "/" + MenuContainerStruct.CurrentBeatmapOsuFileString)

				currentHitObjectStats = CurrentHitObjectStatsString()
				j, err := ioutil.ReadFile(fullPathToOsu) // possibe file open exc
				if err != nil {
					//	fmt.Println("osu file was not found2")
				}
				osuFileStdIN = string(j)
				if strings.Contains(osuFileStdIN, "[HitObjects]") == true {
					splitted := strings.Split(osuFileStdIN, "[HitObjects]")[1]
					newline := strings.Split(splitted, "\n")

					for i := 0; i < len(newline); i++ { //TODO: Add proper exception handler
						if len(newline[i]) > 1 {
							elements := strings.Split(newline[i], ",")[2]
							elementsInt := cast.ToInt(elements)
							ourTime = append(ourTime, elementsInt)
						}
					}
				}

				if strings.HasSuffix(fullPathToOsu, ".osu") == true {
					var bgString string = CurrentBeatmapBackgroundString()
					if bgString != "-5" {
						innerBGPath = MenuContainerStruct.CurrentBeatmapFolderString + "/" + bgString
					}

				} else {
					fmt.Println("skipping bg reloading")
				}
			}
			if osuStatusValue != 2 {
				tempCounter = 0
			}

			if strings.HasSuffix(fullPathToOsu, ".osu") == true && osuStatus == 4 || osuStatus == 5 {

				ppMods = ModsResolver(cast.ToUint32(MenuContainerStruct.CurrentAppliedMods))
				ppSS = PPSS()
				pp99 = PP99()
				pp98 = PP98()
				pp97 = PP97()
				pp96 = PP96()
				pp95 = PP95()
				// tempCurrentAppliedMods = MenuContainerStruct.CurrentAppliedMods
			} else if strings.HasSuffix(fullPathToOsu, ".osu") == true && osuStatus == 2 && tempCounter <= 5 {
				tempCounter++
				ppMods = ModsResolver(cast.ToUint32(MenuContainerStruct.CurrentAppliedMods))
				ppSS = PPSS()
				pp99 = PP99()
				pp98 = PP98()
				pp97 = PP97()
				pp96 = PP96()
				pp95 = PP95()
			}
			group := EverythingInMenu2{
				P: PlayContainerStruct,
				D: MenuContainerStruct,
			}
			jsonByte, err = json.Marshal(group)
			if err != nil {
				fmt.Println("error:", err)
			}
			ws.WriteMessage(1, []byte(jsonByte)) //sending data to the client

			//if err != nil {
			//	log.Println(err)
			//}
			//fmt.Println(countOpenFiles())
			time.Sleep(time.Duration(updateTime) * time.Millisecond)

		}
	} else {
		for {
			ws.WriteMessage(1, []byte(jsonByte))
			time.Sleep(time.Duration(updateTime) * time.Millisecond)
		}

	}
}
func setupRoutes() {
	http.HandleFunc("/ws", wsEndpoint)
}

func countOpenFiles() int64 {
	out, err := exec.Command("/bin/sh", "-c", fmt.Sprintf("lsof -p %v", os.Getpid())).Output()
	if err != nil {
		fmt.Println(err.Error())
	}
	lines := strings.Split(string(out), "\n")
	return int64(len(lines) - 1)
}
func main() {
	// doUpdate()
	if runtime.GOOS == "windows" {
		fmt.Println("Hello from Windows, Please add a browser source in obs to http://127.0.0.1:24050 or refresh the page if you already did that.")
		operatingSystem = 1
	}
	if runtime.GOOS == "linux" {
		fmt.Println("Hello from Linux, Please add a browser source in obs to http://127.0.0.1:24050 or refresh the page if you already did that.")
		operatingSystem = 2

	}

	// if operatingSystem == 2 { // hack to fix "Too many open files"
	// 	var rLimit syscall.Rlimit
	// 	err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rLimit)
	// 	if err != nil {
	// 		fmt.Println("Error Getting Rlimit ", err)
	// 	}
	// 	fmt.Println(rLimit)
	// 	rLimit.Max = 999999
	// 	rLimit.Cur = 999999
	// 	err = syscall.Setrlimit(syscall.RLIMIT_NOFILE, &rLimit)
	// 	if err != nil {
	// 		fmt.Println("Error Setting Rlimit ", err)
	// 	}
	// 	err = syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rLimit)
	// 	if err != nil {
	// 		fmt.Println("Error Getting Rlimit ", err)
	// 	}
	// 	fmt.Println("Rlimit Final", rLimit)
	// }

	path := flag.String("path", "auto", "Path to osu! Songs directory ex: C:\\Users\\BlackShark\\AppData\\Local\\osu!\\Songs")
	updateTimeAs := flag.Int("update", 50, "How fast should we update the values? (in milliseconds)")
	flag.Parse()
	updateTime = *updateTimeAs
	workingDirectory = *path
	if workingDirectory == "auto" {
		fmt.Println("Will try to find Songs folder... (auto flag is set in --path) ")
	}
	go HTTPServer()
	setupRoutes()
	log.Fatal(http.ListenAndServe(":8085", nil))
}
func restart() {
	isRunning = 0
	proc, procerr = kiwi.GetProcessByFileName("osu!.exe")
	for procerr != nil {
		reqRestart = 1
		fmt.Println("reqRestart = 1")
		log.Println("is osu! running? (osu! process was not found, waiting...)")
		proc, procerr = kiwi.GetProcessByFileName("osu!.exe")
		time.Sleep(1 * time.Second)
	}
	if isRunning == 0 {
		fmt.Println("Client Connected, please go to song select and check this console back.")
		StaticOsuStatusAddr := OsuStatusAddr() //we should only check for this address once.
		osuStatusOffset, err := proc.ReadUint32(StaticOsuStatusAddr - 0x4)
		for err != nil {
			log.Println("osu!status offset was not found, retrying...")
			StaticOsuStatusAddr = OsuStatusAddr()
			osuStatusOffset, err = proc.ReadUint32(StaticOsuStatusAddr - 0x4)
			time.Sleep(250 * time.Millisecond)
		}
		uintptrOsuStatus = uintptr(osuStatusOffset)
		osuStatusValue, err := proc.ReadUint16(uintptrOsuStatus)
		if err != nil {
			log.Println("osu!status value was not found, are you sure that osu!stable is running? If so, please report this to GitHub!")
		}

		for {
			fmt.Println("please go to songselect or playmode in order to proceed!")
			osuStatusValue, err = proc.ReadUint16(uintptrOsuStatus)
			if err != nil {
				fmt.Println("It looks like we lost the process, performing a restart...")
				restart()
			}
			if osuStatusValue == 2 {
				break
			}
			if osuStatusValue == 5 {
				break
			}
			time.Sleep(500 * time.Millisecond)

			// time.Sleep(1 * time.Second)

		}
		InitBaseStuff()
		isRunning = 1
		if CurrentPlayTime() == -1 {
			fmt.Println("Failed to get the correct offsets, retrying...")
			restart()

		}

		fmt.Println("it seems that we got the correct offsets, you are good to go!")
	}

}

func firstN(s string, n int) string {
	if len(s) > n {
		return s[:n]
	}
	return s
}

func CurrentBeatmapID() uint32 { //currentbeatmapdata

	currentBeatmapID, err := proc.ReadUint32(uintptr(currentBeatmapDataFirtLevel + 0xC4))
	if err != nil {
		//log.Println("CurrentBeatmapID result pointer failure")
		return 0
	}
	return currentBeatmapID
}
func CurrentBeatmapSetID() uint32 {
	currentSetBeatmapID, err := proc.ReadUint32(uintptr(currentBeatmapDataFirtLevel + 0xC8))
	if err != nil {
		//log.Println("CurrentBeatmapSetID result pointer failure")
		return 0
	}
	return currentSetBeatmapID
}
func CurrentBeatmapString() string {
	beatmapStringSecondLevel, err := proc.ReadUint32(uintptr(currentBeatmapDataFirtLevel + 0x7C))
	if err != nil {
		//log.Println("BeatMapString Second level pointer failure")
		return "-4"
	}

	beatmapStringResult, err := proc.ReadNullTerminatedUTF16String((uintptr(beatmapStringSecondLevel + 0x8)))
	if err != nil {
		//	log.Println("BeatMapString Third level pointer failure")
		return "-5"
	}
	return beatmapStringResult
}

func CurrentBeatmapBackgroundString() string {
	beatmapStringSecondLevel, err := proc.ReadUint32(uintptr(currentBeatmapDataFirtLevel + 0x68))
	if err != nil {
		//log.Println("CurrentBeatmapBackgroundString Second level pointer failure")
		return "-4"
	}

	beatmapStringResult, err := proc.ReadNullTerminatedUTF16String((uintptr(beatmapStringSecondLevel + 0x8)))
	if err != nil {
		//	log.Println("CurrentBeatmapBackgroundString Third level pointer failure")
		return "-5"
	}
	return beatmapStringResult
}

func CurrentBeatmapFolderString() string {

	beatmapFolderStringSecondLevel, err := proc.ReadUint32(uintptr(currentBeatmapDataFirtLevel + 0x74))
	if err != nil {
		//	log.Println("BeatMapFolderString Second level pointer failure")
		return "-4"
	}
	beatmapStringResult, err := proc.ReadNullTerminatedUTF16String((uintptr(beatmapFolderStringSecondLevel + 0x8)))
	if err != nil {
		//	log.Println("BeatMapFolderString Third level pointer failure")
		return "-5"
	}
	return beatmapStringResult
}
func CurrentBeatmapOsuFileString() string {

	beatmapFolderStringSecondLevel, err := proc.ReadUint32(uintptr(currentBeatmapDataFirtLevel + 0x8C))
	if err != nil {
		//log.Println("BeatMapOsuFileString Second level pointer failure")
		return "-4"
	}
	beatmapStringResult, err := proc.ReadNullTerminatedUTF16String((uintptr(beatmapFolderStringSecondLevel + 0x8)))
	if err != nil {
		//	log.Println("BeatMapString Third level pointer failure")
		return "-5"
	}
	// beatmapString := string(beatmapStringResult)
	// beatmapValidString := strings.ToValidUTF8(beatmapString, "")
	return beatmapStringResult
}
func CurrentBeatmapAR() float32 {
	currentSetBeatmapID, err := proc.ReadFloat32(uintptr(currentBeatmapDataFirtLevel + 0x2C))
	if err != nil {
		//	log.Println("AR result level pointer failure")
		return -5
	}
	return currentSetBeatmapID
}
func CurrentBeatmapCS() float32 {
	currentSetBeatmapID, err := proc.ReadFloat32(uintptr(currentBeatmapDataFirtLevel + 0x30))
	if err != nil {
		//	log.Println("CS result level pointer failure")
		return -4
	}
	return currentSetBeatmapID
}
func CurrentBeatmapHP() float32 {
	currentSetBeatmapID, err := proc.ReadFloat32(uintptr(currentBeatmapDataFirtLevel + 0x34))
	if err != nil {
		//	log.Println("HP result level pointer failure")
		return -5
	}
	return currentSetBeatmapID
}
func CurrentBeatmapOD() float32 {
	currentSetBeatmapID, err := proc.ReadFloat32(uintptr(currentBeatmapDataFirtLevel + 0x38))
	if err != nil {
		//	log.Println("OD result level pointer failure")
		return -5
	}
	return currentSetBeatmapID
}

// ------------------- PlayContainer
func CurrentAppliedMods() int32 {
	if osuStatus != 2 {
		return inMenuAppliedModsValue()
	}
	currentCombo, err := proc.ReadInt32(uintptr(playContainer38 + 0x1C))
	if err != nil {
		//		log.Println("CurrentCombo result pointer failure")
		return -5
	}
	xorVal1, err := proc.ReadInt32(uintptr(currentCombo + 0xC))
	if err != nil {
		//	log.Println("CurrentCombo result pointer failure")
		return -6
	}
	xorVal2, err := proc.ReadInt32(uintptr(currentCombo + 0x8))
	if err != nil {
		//	log.Println("CurrentCombo result pointer failure")
		return -7
	}
	val := xorVal2 ^ xorVal1
	return val
}
func CurrentCombo() int32 {
	if osuStatus != 2 {
		return -1
	}

	currentCombo, err := proc.ReadInt32(uintptr(playContainer38 + 0x90))
	if err != nil {
		//	log.Println("CurrentCombo result pointer failure")
		return -5
	}
	return currentCombo
}

func CurrentMaxCombo() int32 {
	if osuStatus != 2 {
		return -1
	}

	currentCombo, err := proc.ReadInt32(uintptr(playContainer38 + 0x68))
	if err != nil {
		//		log.Println("CurrentCombo result pointer failure")
		return -5
	}
	return currentCombo
}
func CurrentHit100c() int16 {
	if osuStatus != 2 {
		return -1
	}

	currentCombo, err := proc.ReadInt16(uintptr(playContainer38 + 0x84)) //2 bytes
	if err != nil {
		//	log.Println("CurrentHit100c result pointer failure")
		return -5
	}
	return currentCombo
}
func CurrentHit300c() int16 {
	if osuStatus != 2 {
		return -1
	}
	current300, err := proc.ReadInt16(uintptr(playContainer38 + 0x86)) //2 bytes
	if err != nil {
		//		log.Println("CurrentHit300c result pointer failure")
		return -5
	}
	//currentgeki, err := proc.ReadInt16(uintptr(comboSecondLevel + 0x8A)) //2 bytes
	//current300Result := current300 + currentgeki // thats not how the game works /shrug
	return current300
}
func CurrentHit50c() int16 {
	if osuStatus != 2 {
		return -1
	}

	currentCombo, err := proc.ReadInt16(uintptr(playContainer38 + 0x88)) //2 bytes
	if err != nil {
		//	log.Println("CurrentHitMiss result pointer failure")
		return -5
	}
	return currentCombo
}
func inMenuAppliedModsValue() int32 {
	inMenuAppliedModsFirstLevel, err := proc.ReadInt32(inMenuAppliedModsBase)
	if err != nil {
		//	log.Println("CurrentHitMiss result pointer failure")
		return -1
	}
	inMenuAppliedModsResult, err := proc.ReadInt32(uintptr(inMenuAppliedModsFirstLevel))
	if err != nil {
		//	log.Println("CurrentHitMiss result pointer failure")
		return -5
	}
	return inMenuAppliedModsResult
}
func CurrentHitObjectStatsString() string {
	CurrentHitObjectStatsStringFirstLevel, err := proc.ReadInt32(uintptr(bpmBase))
	if err != nil {
		//	log.Println("CurrentHitMiss result pointer failure")
		return "-1"
	}
	CurrentHitObjectStatsStringSecondLevel, err := proc.ReadInt32(uintptr(CurrentHitObjectStatsStringFirstLevel + 0x4))
	if err != nil {
		//	log.Println("CurrentHitMiss result pointer failure")
		return "-2"
	}
	CurrentHitObjectStatsStringThirdLevel, err := proc.ReadInt32(uintptr(CurrentHitObjectStatsStringSecondLevel + 0x20))
	if err != nil {
		//	log.Println("CurrentHitMiss result pointer failure")
		return "-3"
	}
	CurrentHitObjectStatsStringFourthLevel, err := proc.ReadInt32(uintptr(CurrentHitObjectStatsStringThirdLevel + 0x2F4))
	if err != nil {
		//	log.Println("CurrentHitMiss result pointer failure")
		return "-4"
	}
	CurrentHitObjectStatsStringResult, err := proc.ReadNullTerminatedUTF16String(uintptr(CurrentHitObjectStatsStringFourthLevel + 0x8))
	if err != nil {
		//	log.Println("CurrentHitMiss result pointer failure")
		return "-5"
	}
	return CurrentHitObjectStatsStringResult
}
func CurrentHitMiss() int16 {
	if osuStatus != 2 {
		return -1
	}

	currentCombo, err := proc.ReadInt16(uintptr(playContainer38 + 0x8E)) //2 bytes
	if err != nil {
		//		log.Println("CurrentHitMiss result pointer failure")
		return -5
	}
	return currentCombo
}
func CurrentScore() int32 {
	if osuStatus != 2 {
		return -1
	}

	currentCombo, err := proc.ReadInt32(uintptr(playContainer38 + 0x74))
	if err != nil {
		//	log.Println("CurrentScore result pointer failure")
		return -5
	}
	return currentCombo
}
func CurrentGameMode() int32 {
	if osuStatus != 2 {
		return -1
	}

	currentCombo, err := proc.ReadInt32(uintptr(playContainer38 + 0x64))
	if err != nil {
		//		log.Println("GameMode result pointer failure")
		return -5
	}
	return currentCombo
}
func CurrentAccuracy() float64 {
	if osuStatus != 2 {
		return -1
	}

	comboSecondLevel, err := proc.ReadUint32(uintptr(playContainerFirstlevel) + 0x48)
	if err != nil {
		//	log.Println("Accuracy Second level pointer failure")
		return -4
	}
	currentCombo, err := proc.ReadFloat64(uintptr(comboSecondLevel + 0x14))
	if err != nil {
		//		log.Println("Accuracy result pointer failure")
		return -5
	}
	return currentCombo
}
func CurrentPlayerHP() int8 {
	if osuStatus != 2 {
		return -1
	}

	comboSecondLevel, err := proc.ReadUint32(uintptr(playContainerFirstlevel) + 0x40)
	if err != nil {
		//		log.Println("CurrentPlayerHP Second level pointer failure")
		return -4
	}
	currentCombo, err := proc.ReadInt8(uintptr(comboSecondLevel + 0x1C))
	if err != nil {
		//	log.Println("CurrentPlayerHP result pointer failure")
		return -5
	}
	return currentCombo
}
func CurrentPlayerHPSmoothed() int8 {
	if osuStatus != 2 {
		return -1
	}

	comboSecondLevel, err := proc.ReadUint32(uintptr(playContainerFirstlevel) + 0x40)
	if err != nil {
		//	log.Println("CurrentPlayerHPSmoothed Second level pointer failure")
		return -4
	}
	currentCombo, err := proc.ReadInt8(uintptr(comboSecondLevel + 0x14))
	if err != nil {
		//		log.Println("CurrentPlayerHPSmoothed result pointer failure")
		return -5
	}
	return currentCombo
}
func CurrentPlayTime() int32 {
	playTimeFirstLevel, err := proc.ReadUint32(playTime)
	if err != nil {
		//	log.Println("playTime Base level failure")
		return -1
	}
	playTimeValue, err := proc.ReadUint32(uintptr(playTimeFirstLevel))
	if err != nil {
		//	log.Println("playTime Result level failure")
		return -1
	}

	return cast.ToInt32(playTimeValue)
}

func ResolveSongsFolder() string {

	songsInitAddr, err := proc.ReadUint32(songsFolderAddr - 0x4)
	if err != nil {
		//	log.Println("playTime Base level failure")
		return "-1"
	}
	songsBase, err := proc.ReadUint32(uintptr(songsInitAddr + 0x34))
	if err != nil {
		//	log.Println("playTime Base level failure")
		return "-2"
	}
	songsFirstLevel, err := proc.ReadUint32(uintptr(songsBase + 0x10))
	if err != nil {
		//	log.Println("playTime Base level failure")
		return "-3"
	}

	songsResult, err := proc.ReadNullTerminatedUTF16String(uintptr(songsFirstLevel + 0x20))

	if err != nil {
		//	log.Println("playTime Base level failure")
		return "-5"
	}
	return songsResult
}
func PP() string {
	if operatingSystem == 1 {
		calc, err := exec.Command("deps/oppai.exe", fullPathToOsu, "-end"+lastObject, ppAcc+"%", ppCombo+"x", ppMiss+"m", pp100+"x100", pp50+"x50", "+"+ppMods, "-ojson").Output()
		if err != nil {
			//fmt.Println(err)
		}

		return strings.ToValidUTF8(cast.ToString(calc), "")
	} else {
		calc := Cmd("oppai"+" "+"\""+fullPathToOsu+"\""+" "+"-end"+lastObject+" "+ppAcc+"%"+" "+ppCombo+"x"+" "+ppMiss+"m"+" "+pp100+"x100"+" "+pp50+"x50"+" "+"+"+ppMods+" "+"-ojson", true)
		//calc := Cmd("oppai"+" "+"\""+fullPathToOsu+"\""+" "+"-end"+lastObject+" "+ppAcc+"%"+" "+ppCombo+"x"+" "+ppMiss+"m"+" "+pp100+"x100"+" "+pp50+"x50"+" "+" "+"-ojson", true)
		return strings.ToValidUTF8(cast.ToString(calc), "")
	}

}
func PPifFC() string {
	if operatingSystem == 1 {
		calc, err := exec.Command("deps/oppai.exe", fullPathToOsu, ppAcc+"%", pp100+"x100", pp50+"x50", "+"+ppMods, "-ojson").Output()
		if err != nil {
			//fmt.Println(err)
		}

		return strings.ToValidUTF8(cast.ToString(calc), "")
	} else {
		calc := Cmd("oppai"+" "+"\""+fullPathToOsu+"\""+" "+" "+ppAcc+"%"+" "+pp100+"x100"+" "+pp50+"x50"+" "+"+"+ppMods+" "+"-ojson", true)

		return strings.ToValidUTF8(cast.ToString(calc), "")
	}

}
func PPSS() string {
	if operatingSystem == 1 {
		calc, err := exec.Command("deps/oppai.exe", fullPathToOsu, "100%", "+"+ppMods, "-ojson").Output()
		if err != nil {
			//fmt.Println(err)
		}

		return strings.ToValidUTF8(cast.ToString(calc), "")
	} else {
		calc := Cmd("oppai"+" "+"\""+fullPathToOsu+"\""+" "+"100%"+" "+"+"+ppMods+" "+"-ojson", true)

		return strings.ToValidUTF8(cast.ToString(calc), "")
	}

}
func PP99() string {
	if operatingSystem == 1 {
		calc, err := exec.Command("deps/oppai.exe", fullPathToOsu, "99%", "+"+ppMods, "-ojson").Output()
		if err != nil {
			//fmt.Println(err)
		}

		return strings.ToValidUTF8(cast.ToString(calc), "")
	} else {
		calc := Cmd("oppai"+" "+"\""+fullPathToOsu+"\""+" "+"99%"+" "+"+"+ppMods+" "+"-ojson", true)

		return strings.ToValidUTF8(cast.ToString(calc), "")
	}

}
func PP98() string {
	if operatingSystem == 1 {
		calc, err := exec.Command("deps/oppai.exe", fullPathToOsu, "98%", "+"+ppMods, "-ojson").Output()
		if err != nil {
			//fmt.Println(err)
		}

		return strings.ToValidUTF8(cast.ToString(calc), "")
	} else {
		calc := Cmd("oppai"+" "+"\""+fullPathToOsu+"\""+" "+"98%"+" "+"+"+ppMods+" "+"-ojson", true)

		return strings.ToValidUTF8(cast.ToString(calc), "")
	}

}
func PP97() string {
	if operatingSystem == 1 {
		calc, err := exec.Command("deps/oppai.exe", fullPathToOsu, "97%", "+"+ppMods, "-ojson").Output()
		if err != nil {
			//fmt.Println(err)
		}

		return strings.ToValidUTF8(cast.ToString(calc), "")
	} else {
		calc := Cmd("oppai"+" "+"\""+fullPathToOsu+"\""+" "+"97%"+" "+"+"+ppMods+" "+"-ojson", true)

		return strings.ToValidUTF8(cast.ToString(calc), "")
	}

}
func PP96() string {
	if operatingSystem == 1 {
		calc, err := exec.Command("deps/oppai.exe", fullPathToOsu, "96%", "+"+ppMods, "-ojson").Output()
		if err != nil {
			//fmt.Println(err)
		}

		return strings.ToValidUTF8(cast.ToString(calc), "")
	} else {
		calc := Cmd("oppai"+" "+"\""+fullPathToOsu+"\""+" "+"96%"+" "+"+"+ppMods+" "+"-ojson", true)

		return strings.ToValidUTF8(cast.ToString(calc), "")
	}

}
func PP95() string {
	if operatingSystem == 1 {
		calc, err := exec.Command("deps/oppai.exe", fullPathToOsu, "95%", "+"+ppMods, "-ojson").Output()
		if err != nil {
			// fmt.Println(err)
		}

		return strings.ToValidUTF8(cast.ToString(calc), "")
	} else {
		calc := Cmd("oppai"+" "+"\""+fullPathToOsu+"\""+" "+"95%"+" "+"+"+ppMods+" "+"-ojson", true)

		return strings.ToValidUTF8(cast.ToString(calc), "")
	}

}

func SliceIndex(limit int, predicate func(i int) bool) int {
	for i := 0; i < limit; i++ {
		if predicate(i) {
			return i
		}
	}

	return -1
}

func ModsResolver(xor uint32) string {
	return Mods(xor).String()
}
func HTTPServer() {

	fs := http.FileServer(http.Dir("static"))
	http.Handle("/", fs)
	//box := packr.NewBox("./index")
	// http.Handle("/Songs/", http.StripPrefix("/Songs/", http.FileServer(http.Dir(workingDirectory))))
	//	http.Handle("/", http.FileServer(box))
	http.HandleFunc("/json", handler)
	http.ListenAndServe(":24050", nil)
}
func handler(w http.ResponseWriter, r *http.Request) {
	if isRunning == 1 {
		fmt.Fprintf(w, cast.ToString(jsonByte))

	} else {
		fmt.Fprintf(w, "osu! is not fully loaded!")
	}

}

type mmap struct {
	Start    uintptr
	End      uintptr
	Perms    string
	Offset   uintptr
	DevMajor int
	DevMinor int
	Inode    int
	Path     string
}

func (m *mmap) Size() uintptr {
	return m.End - m.Start
}

func scan(f *os.File, maps []mmap, pattern string) (uint32, error) {
	var largestMap uintptr
	for _, amap := range maps {
		if amap.Perms[0] != 'r' || amap.Perms[2] != 'x' {
			continue
		}
		if amap.Size() > largestMap {
			largestMap = amap.Size()
		}
	}
	pat, err := parsePattern(pattern)
	if err != nil {
		return 0, err
	}
	buf := make([]byte, largestMap)
	for _, amap := range maps {
		if amap.Perms[0] != 'r' || amap.Perms[2] != 'x' {
			continue
		}
		size := amap.Size()
		_, err := f.Seek(int64(amap.Start), 0)
		if err != nil {
			return 0, err
		}
		_, err = io.ReadFull(f, buf[0:size])
		if err != nil {
			continue
		}
		needle := pat.Bytes[0]
		mask := pat.Mask[0]
		var j uintptr
	outer:
		for j = 0; (j + 4) < size; j += 1 {
			haystack := binary.LittleEndian.Uint32(buf[j : j+4])
			if needle^haystack&mask == 0 {
				for k := range pat.Bytes {
					needle := pat.Bytes[k]
					mask := pat.Mask[k]
					haystack := binary.LittleEndian.Uint32(
						buf[j+uintptr(4*k) : j+4+uintptr(4*k)])
					if needle^haystack&mask != 0 {
						continue outer
					}
				}
				return uint32(amap.Start) + uint32(j), nil
			}
		}
	}
	return 0, errPatternNotFound
}

func readMaps(pid int) ([]mmap, error) {
	f, err := os.Open(fmt.Sprintf("/proc/%d/maps", pid))
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var maps []mmap
	s := bufio.NewScanner(f)
	for s.Scan() {
		var amap mmap
		_, err := fmt.Sscanf(
			s.Text(), "%x-%x %s %x %x:%x %d %s",
			&amap.Start, &amap.End, &amap.Perms, &amap.Offset,
			&amap.DevMajor, &amap.DevMinor, &amap.Inode, &amap.Path,
		)
		if err != nil && err != io.EOF {
			continue
		}
		maps = append(maps, amap)
	}
	return maps, nil
}

type pattern struct {
	Bytes []uint32
	Mask  []uint32
}

func parsePattern(s string) (*pattern, error) {
	var bytes, mask []byte
	for _, bytestr := range strings.Split(s, " ") {
		if bytestr == "??" {
			bytes = append(bytes, 0x00)
			mask = append(mask, 0x00)
			continue
		}
		b, err := strconv.ParseUint(bytestr, 16, 8)
		if err != nil {
			return nil, err
		}
		bytes = append(bytes, byte(b))
		mask = append(mask, 0xFF)
	}
	var p pattern
	for i := 0; i < len(bytes); i += 4 {
		var byt, mas uint32
		for i, b := range bytes[i : i+4] {
			byt |= uint32(b) << (i * 8)
		}
		for i, m := range mask[i : i+4] {
			mas |= uint32(m) << (i * 8)
		}
		p.Bytes = append(p.Bytes, byt)
		p.Mask = append(p.Mask, mas)
	}
	return &p, nil
}

var (
	errNoPIDMatched = errors.New("No PID matched the criteria")
)

var (
	errPatternNotFound = errors.New("Pattern not found")
)

func doUpdate(url string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	err = update.Apply(resp.Body, update.Options{})
	if err != nil {
		// error handling
	}
	return err
}
