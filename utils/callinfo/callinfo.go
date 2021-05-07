
package callinfo

import (
        "fmt"
        "runtime"
        "strings"
	"strconv"

	"path/filepath"
)

func Goid() int {
	defer func()  {
		if err := recover(); err != nil {
			fmt.Println("panic recover:panic info:%v", err)     }
	}()
 
	var buf [64]byte
	n := runtime.Stack(buf[:], false)
	idField := strings.Fields(strings.TrimPrefix(string(buf[:n]), "goroutine "))[0]
	id, err := strconv.Atoi(idField)
	if err != nil {
		panic(fmt.Sprintf("cannot get goroutine id: %v", err))
	}
	
	return id
}

func Prefix() string {
        var prefix string  

	// Ask runtime.Callers for up to 10 pcs, including runtime.Callers itself.
	pc := make([]uintptr, 10)
	n := runtime.Callers(0, pc)
	if n == 0 {
		// No pcs available. Stop now.
		// This can happen if the first argument to runtime.Callers is large.
		return "" 
	}

	pc = pc[:n] // pass only valid pcs to runtime.CallersFrames
	frames := runtime.CallersFrames(pc)

	// Loop to get frames.
	// A fixed number of pcs can expand to an indefinite number of Frames.
	for {
		_, more := frames.Next()
                prefix += " "  

		if !more {
			break
		}
	}

        return prefix
}


func Stacks() []string {
        var nilresult = make([]string, 0)  

	// Ask runtime.Callers for up to 32 pcs, including runtime.Callers itself.
	pc := make([]uintptr, 32)
	n := runtime.Callers(0, pc)
	if n == 0 {
		// No pcs available. Stop now.
		// This can happen if the first argument to runtime.Callers is large.
		return nilresult 
	}

	pc = pc[:n] // pass only valid pcs to runtime.CallersFrames
	frames := runtime.CallersFrames(pc)

	// Loop to get frames.
	// A fixed number of pcs can expand to an indefinite number of Frames.
	stacks := make([]runtime.Frame, 0, 32) 
	var foundCaller bool
	for {
		f, more := frames.Next()
                //prefix += " "  
                //log.Debugf("%s:%d %s\n", f.File, f.Line, f.Function)
                if !foundCaller && strings.HasSuffix(f.File, "callinfo.go") && more {
                        f, _ = frames.Next()
                        //relFuncName := strings.TrimPrefix(f.Function, "github.com/siegfried415/go-crawling-bazaar/")
                        //caller := fmt.Sprintf("%s:%d %s", filepath.Base(f.File), f.Line, relFuncName)
                        foundCaller = true
                }
                if foundCaller {
                        stacks = append(stacks, f)
                }

		if !more {
			break
		}
	}

        if len(stacks) > 0 {
		stacksStr := make([]string, 0, len(stacks))
		for i, s := range stacks {
			if s.Line > 0 {
				fName := strings.TrimPrefix(s.Function, "github.com/siegfried415/go-crawling-bazaar/")
				stackStr := fmt.Sprintf("#%d %s@%s:%d     ", i, fName, filepath.Base(s.File), s.Line)
				stacksStr = append(stacksStr, stackStr)
			}
		}
		return stacksStr
        }

        return nilresult 
}
