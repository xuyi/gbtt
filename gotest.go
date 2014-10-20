package main

import (
    "os"
    "io/ioutil"
    "net"
    "fmt"
    "flag"
    "regexp"
    "time"
    "log"
)


var (
    ip = flag.String("h", "127.0.0.1", "server ip")
    port = flag.Int("p", -1, "server port")
    times = flag.Int("n", 1, "test times")
    timeout = flag.Int("t", 5, "timeout")

    testfile = flag.String("f", "", "protocol file(data first)")
    testdata = flag.String("d", "", "protocol data")
    match = flag.String("m", "", "response match")

    protocol = flag.String("protocol", "tcp", "server protocol")
    debug = flag.Bool("debug", false, "open debug")

	flagQps = flag.Int("q", 0, "qps")
)

func testHandler(r *regexp.Regexp, content []byte, result chan int) {
    conn, err := net.Dial(*protocol, fmt.Sprintf("%s:%d", *ip, *port))
    if err != nil {
        // fatalExit(-1, "connect error")
        result <- -1
        return
    }
    defer conn.Close()
    _, ok := conn.Write(content)
    if ok != nil {
        result <- -1
        return
    }

    reply := make([]byte, 2048)
    readlen, ok := conn.Read(reply)
    if ok != nil {
        result <- -1
        return
    }

    if *debug {
        log.Println("connect:", fmt.Sprintf("%s:%d", *ip, *port))
        log.Println("respone:", string(reply[0:readlen]))
    }
    if !r.MatchString(string(reply)) {
        result <- -1
    } else {
        result <- 0
    }
}

func fatalExit(code int, msg string) {
    // flag.Usage()
    fmt.Printf("result=0&code=%d&msg=%s&end=0", code, msg)
    os.Exit(code)
}

func succExit(code int, msg string) {
    fmt.Printf("result=0&code=%d&msg=%s&end=0", code, msg)
    os.Exit(code)
}

func errorExit(code int, msg string) {
    fmt.Println(msg)
    os.Exit(code)
}

func main() {
    flag.Parse()

    if *ip == "" || *port == -1 {
        fatalExit(-1, "ip or port error")
    }

    var testContent []byte

    if *testfile == "" && *testdata == "" {
        fatalExit(-1, "test content error")
    } else if *testdata != "" {
        testContent = []byte(*testdata + "\n")
    } else{
        f, err := os.Open(*testfile)
        if err != nil {
            fatalExit(-1, "protocol data error")
        }
        content, err := ioutil.ReadAll(f)
        testContent = content
        if err != nil {
            fatalExit(-1, "protocol data error")
        }
    }

    if *match == "" {
        fatalExit(-1, "match content error")
    }
    r, err := regexp.Compile(*match)
    if err != nil {
        fatalExit(-1, "match content error")
    }

    result := make(chan int)

	qps := *flagQps
	var throttle <-chan time.Time
	if qps > 0 {
		throttle = time.Tick(time.Duration(1e6/(qps)) * time.Microsecond)
	}

    for i:=0 ;i<*times;i++{
        go func(){
			testHandler(r, testContent, result)
		}()
		if qps > 0 {
			<-throttle
		}
    }

    testNum := 0
    isTimeout := false
    kk := 0

    for {
        select {
        case r := <-result :
            kk += 1
            if r == 0{
                testNum += 1
            }
        case <-time.After(time.Second * time.Duration(*timeout)):
            isTimeout = true
        }
        if isTimeout || kk == *times {
            break
        }
    }

    rstMsg := fmt.Sprintf("test:%d,succ:%d,fail:%d", *times, testNum, *times - testNum)
    if testNum !=  *times {
        fatalExit(-1, rstMsg)
    } else {
        succExit(0, rstMsg)
    }
}
