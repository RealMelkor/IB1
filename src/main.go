package main

import (
	"log"
	"IB1/web"
	"IB1/db"
	"math/rand"
	"time"
)

func randString() string {
	letters := "abcdefghijklmnopqrstuvwxyz"
	length := rand.Intn(20) + 5
	str := ""
	for i := 0; i < length; i++ {
		str += string(letters[rand.Intn(len(letters))])
	}
	return str
}

func floodPosts(thread db.Thread) {
	posts := int64(0)
	start := time.Now().Unix()
	for {
		if _, err := db.CreatePost(thread, randString(), "", nil); err != nil {
			log.Println(err)
			break
		}
		posts += 1
		if posts % 100 == 0 {
			log.Println(thread.ID,
				posts / (time.Now().Unix() - start + 1),
				"posts/second")
		}
	}
}

func main() {

	if err := db.Init(); err != nil {
		log.Println(err)
		return
	}

	//a, _ := db.GetBoard("a")
	//b, _ := db.GetBoard("b")

	//go floodPosts(a.Threads[0])
	//floodPosts(b.Threads[0])

	if err := web.Init(); err != nil {
		log.Println(err)
		return
	}
}
