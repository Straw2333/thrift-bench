package main

import (
	example "demo-server/kitex_gen/example/userservice"
	"log"
)

func main() {
	svr := example.NewServer(new(UserServiceImpl))

	err := svr.Run()

	if err != nil {
		log.Println(err.Error())
	}
}
