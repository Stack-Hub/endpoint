package main

import (
    "C"
    "fmt"
    "strconv"
    "unsafe"
    "strings"
    "os"

    "github.com/rainycape/dl"
)

// go build -buildmode=c-shared -o listener.so listener.go
func main() {}

//export write
func write(fd C.int, buf uintptr, num C.int) int32 {
    lib, err := dl.Open("libc", 0)
    if err != nil {
        return 0
    }
    defer lib.Close()

    var realwrite func(fd C.int, buf uintptr, num C.int) int32
    lib.Sym("write", &realwrite)

    str := C.GoString((*C.char)(unsafe.Pointer(buf)))
    if fd == 2 && strings.HasPrefix(str, "Allocated port") {
        var dynport int
        num, _ := fmt.Sscanf(str, "Allocated port %d for remote forward", &dynport)
        if num == 1 {
            rfwd := strconv.Itoa(dynport)
            os.Setenv("SSH_RFWD", rfwd)
        }        
    }

    return realwrite(fd, buf, num)
}
