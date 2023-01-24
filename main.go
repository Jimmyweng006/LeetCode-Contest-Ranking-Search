package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"
)

// credit to: https://mholt.github.io/json-to-go/
type RankInfoDTO struct {
	Time        float64       `json:"time"`
	IsPast      bool          `json:"is_past"`
	Submissions []interface{} `json:"submissions"`
	Questions   []interface{} `json:"questions"`
	TotalRank   []TotalRank   `json:"total_rank"`
	UserNum     int           `json:"user_num"`
}
type UserBadge struct {
	Icon        string `json:"icon"`
	DisplayName string `json:"display_name"`
}
type TotalRank struct {
	ContestID     int         `json:"contest_id"`
	Username      string      `json:"username"`
	UsernameColor interface{} `json:"username_color"`
	UserBadge     UserBadge   `json:"user_badge"`
	UserSlug      string      `json:"user_slug"`
	CountryCode   string      `json:"country_code"`
	CountryName   string      `json:"country_name"`
	Rank          int         `json:"rank"`
	Score         int         `json:"score"`
	FinishTime    int         `json:"finish_time"`
	GlobalRanking int         `json:"global_ranking"`
	DataRegion    string      `json:"data_region"`
}

const (
	baseURL = "https://leetcode.com/contest/api/ranking/%s/?pagination=%s&region=global"
)

var mu sync.Mutex
var requestCounter = 0
var totalCounter = 0
var contestName string
var username string

func main() {
	// weekly: https://leetcode.com/contest/api/ranking/weekly-contest-329/?pagination=1&region=global
	// biweekly: https://leetcode.com/contest/api/ranking/biweekly-contest-96?pagination=1&region=global
	/*
		weekly-contest-329 Jimmyweng006
	*/
	fmt.Printf("please enter contestName and username, separate by space: ")
	fmt.Scanf("%s %s", &contestName, &username)

	url := fmt.Sprintf(baseURL, contestName, "1")
	fmt.Println("url: ", url)

	rankInfoDTO := getRankInfoDTOByURL(url)

	lastThreadPeople := rankInfoDTO.UserNum % (25 * 100)
	numberOfThread := rankInfoDTO.UserNum / (25 * 100)
	if lastThreadPeople != 0 {
		numberOfThread++
	}
	fmt.Println("numberOfThread:", numberOfThread)
	fmt.Println("lastThreadPeople:", lastThreadPeople)

	// 25000 -> 10 threads
	// thread1 need to handle these pages -> 1 + 0 * offset(100), 2 + 0 * offset(100) ... 100 + 0 * offset(100)
	// thread2 need to handle these pages -> 1 + 1 * offset(100), 2 + 1 * offset(100) ... 100 + 1 * offset(100)

	res := make(chan int)
	for i := 1; i <= numberOfThread; i++ {
		if i != numberOfThread {
			go worker(username, (i-1)*100, 2500, res, i)
		} else {
			go worker(username, (i-1)*100, lastThreadPeople, res, i)
		}
	}

	for i := 0; i < numberOfThread; i++ {
		x := <-res
		fmt.Println("channel val:", x)
		if x != -1 {
			fmt.Println(fmt.Sprintf("found user: %s at %d.", username, x))
			fmt.Println("totalCounter:", totalCounter)
			fmt.Println("main thread end")
			return
		}
	}

	fmt.Println(fmt.Sprintf("user: %s not found.", username))
	fmt.Println("totalCounter:", totalCounter)
	fmt.Println("main thread end")
}

func worker(username string, offset int, numberOfPeople int, res chan int, t int) {
	fmt.Println(fmt.Sprintf("thread t%d start", t))
	numberOfPage := 100
	if numberOfPeople != 2500 {
		numberOfPage = numberOfPeople/25 + 1
	}

	for page := 1; page <= numberOfPage; page++ {
		pageIdx := page + offset
		url := fmt.Sprintf(baseURL, contestName, strconv.Itoa(pageIdx))

		rankInfoDTO := getRankInfoDTOByURL(url)

		ranking := rankInfoDTO.TotalRank
		// should not go to follow case, only for safe
		if len(ranking) == 0 {
			res <- -1
			fmt.Println(fmt.Sprintf("thread t%d end", t))
			return
		}

		fmt.Println(fmt.Sprintf("thread t%d parse ranking, pageIdx: %d", t, pageIdx))
		for i := 1; i <= len(ranking); i++ {
			if ranking[i-1].Username == username {
				res <- ranking[i-1].Rank
				fmt.Println(fmt.Sprintf("thread t%d end", t))
				return
			}
		}

		// avoid rate limit exceed -> cause another issue, the benefit of parallel thread diappear
		mu.Lock()
		requestCounter++
		totalCounter++
		if requestCounter > 300 {
			time.Sleep(30 * time.Second)
			fmt.Println("cur totalCounter:", totalCounter)
			requestCounter -= 300
		}
		mu.Unlock()

		time.Sleep(1000 * time.Millisecond)
	}

	res <- -1
	fmt.Println(fmt.Sprintf("thread t%d end", t))
}

// helper functions
func getRankInfoDTOByURL(url string) RankInfoDTO {
	resp, err := http.Get(url)
	if err != nil {
		log.Fatal(err)
	} else if resp.Status != "200 OK" {
		fmt.Println("cur totalCounter:", totalCounter)
		fmt.Println(fmt.Sprintf("error while getting response from leetcode: %s", resp.Status))
		log.Fatal(err)
	}

	// read response body
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}
	// close response body
	resp.Body.Close()

	rankInfoDTO := RankInfoDTO{}
	err = json.Unmarshal(body, &rankInfoDTO)
	if err != nil {
		fmt.Println(string(body))
		log.Fatal(err)
	}

	return rankInfoDTO
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
