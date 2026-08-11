package main

import (
	"context"
	"log"
	"net/http"

	"connectrpc.com/connect"
	podcastv1 "example.com/connect-demo/gen/podcast/v1"
	"example.com/connect-demo/gen/podcast/v1/podcastv1connect"
)

type podcastService struct {
	podcastv1connect.UnimplementedPodcastServiceHandler
}

func (s *podcastService) ListPodcasts(
	ctx context.Context,
	req *connect.Request[podcastv1.ListPodcastsRequest],
) (*connect.Response[podcastv1.ListPodcastsResponse], error) {
	return connect.NewResponse(&podcastv1.ListPodcastsResponse{
		Podcasts: []*podcastv1.Podcast{
			{Id: "1", Title: "The Daily", Host: "Michael Barbaro"},
		},
	}), nil
}

func main() {
	mux := http.NewServeMux()
	path, handler := podcastv1connect.NewPodcastServiceHandler(&podcastService{})
	mux.Handle(path, handler)
	log.Println("Connect server on :8081")
	log.Fatal(http.ListenAndServe(":8081", mux))
}
