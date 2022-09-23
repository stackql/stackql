package srv

import (
	"bufio"
	"fmt"
	"math/rand"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/stackql/stackql/internal/stackql/driver"
	"github.com/stackql/stackql/internal/stackql/dto"
	"github.com/stackql/stackql/internal/stackql/entryutil"
	"github.com/stackql/stackql/internal/stackql/handler"
	"github.com/stackql/stackql/internal/stackql/iqlerror"

	lrucache "vitess.io/vitess/go/cache"
)

func handleConnection(c net.Conn, runtimeCtx dto.RuntimeCtx, lruCache *lrucache.LRUCache) {
	fmt.Printf("Serving %s\n", c.RemoteAddr().String())
	for {
		netData, err := bufio.NewReader(c).ReadString('\n')

		if err != nil {
			fmt.Println(err)
			return
		}

		temp := strings.TrimSpace(string(netData))
		if temp == "STOP" {
			break
		}
		inputBundle, err := entryutil.BuildInputBundle(runtimeCtx)
		if err != nil {
			fmt.Println(err)
			return
		}
		handlerContext, _ := handler.GetHandlerCtx(netData, runtimeCtx, lruCache, inputBundle)
		handlerContext.Outfile = c
		handlerContext.OutErrFile = c
		defer iqlerror.HandlePanic(c)
		if handlerContext.RuntimeContext.DryRunFlag {
			driver.ProcessDryRun(&handlerContext)
			continue
		}
		driver.ProcessQuery(&handlerContext)
	}
	c.Close()
}

func Serve(portNo int, runtimeCtx dto.RuntimeCtx, lruCache *lrucache.LRUCache) {
	arguments := os.Args
	if len(arguments) == 1 {
		fmt.Println("Please provide a port number!")
		// return
	}

	portStr := strconv.Itoa(portNo)

	l, err := net.Listen("tcp4", ":"+portStr)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer l.Close()
	rand.Seed(time.Now().Unix())

	for {
		c, err := l.Accept()
		if err != nil {
			fmt.Println(err)
			return
		}
		go handleConnection(c, runtimeCtx, lruCache)
	}
}
