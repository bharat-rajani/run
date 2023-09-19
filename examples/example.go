package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/bharat-rajani/rungroup"
	"github.com/bharat-rajani/rungroup/pkg/concurrent"
	"net/http"
	"strconv"
)

type JsonResp map[string]any

const userID = 1

func main() {

	// they key of the map (our goroutine identifier) will be string, and the value will be response from goroutine
	g, ctx := rungroup.WithContextResultMap[string, JsonResp](context.Background(), concurrent.NewRWMutexMap())
	// g, ctx := rungroup.WithContextAndMaps(context.Background(),new(sync.Map)) //refer Benchmarks for performance difference

	/*
		Contrived example where we need to fetch a user details, their posts and their albums.
		We have a user with id as 1
			we need to get the user details
			concurrently we need to get the posts belonging to a particular user
			concurrently we need to get the albums belonging to a particular user
			optionally we need to login the user
	*/

	g.GoWithFunc(FetchUser, ctx, true, "fetch_user")

	g.GoWithFunc(FetchPosts, ctx, true, "fetch_posts")

	g.GoWithFunc(FetchAlbums, ctx, true, "fetch_albums")

	// returns first error from interrupter routine
	err := g.Wait()
	if err != nil {
		fmt.Println(err)
	}

	prettyPrint(g.GetResultByID("fetch_user"))
	prettyPrint(g.GetResultByID("fetch_posts"))
	prettyPrint(g.GetResultByID("fetch_albums"))
}

func FetchUser(ctx context.Context) (JsonResp, error) {

	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("https://jsonplaceholder.typicode.com/users/%d", userID), nil)
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	var tResp JsonResp
	if err := json.NewDecoder(resp.Body).Decode(&tResp); err != nil {
		return nil, err
	}
	return tResp, nil
}

func FetchPosts(ctx context.Context) (JsonResp, error) {

	req, err := http.NewRequest(http.MethodGet, "https://jsonplaceholder.typicode.com/posts", nil)
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	var tResp []JsonResp
	if err := json.NewDecoder(resp.Body).Decode(&tResp); err != nil {
		return nil, err
	}
	posts := make(JsonResp)
	userIDStr := strconv.Itoa(userID)
	for _, t := range tResp {
		if id := t["userId"]; id.(float64) == float64(userID) {
			if posts[userIDStr] == nil {
				posts[userIDStr] = make([]JsonResp, 0)
			}
			delete(t, "userId")
			posts[userIDStr] = append(posts[userIDStr].([]JsonResp), t)
		}
	}
	return posts, nil
}

func FetchAlbums(ctx context.Context) (JsonResp, error) {

	req, err := http.NewRequest(http.MethodGet, "https://jsonplaceholder.typicode.com/albums", nil)
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	var tResp []JsonResp
	if err := json.NewDecoder(resp.Body).Decode(&tResp); err != nil {
		return nil, err
	}

	albums := make(JsonResp)
	userIDStr := strconv.Itoa(userID)
	for _, t := range tResp {
		if id := t["userId"]; id.(float64) == float64(userID) {
			if albums[userIDStr] == nil {
				albums[userIDStr] = make([]JsonResp, 0)
			}
			delete(t, "userId")
			albums[userIDStr] = append(albums[userIDStr].([]JsonResp), t)
		}
	}
	return albums, nil
}

func prettyPrint(V any, err error, ok bool) {
	if err != nil || !ok {
		return
	}
	d, err := json.MarshalIndent(V, "", "    ")
	if err != nil {
		return
	}
	fmt.Println(string(d))
}
