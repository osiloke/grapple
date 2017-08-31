package scraper

import (
	"reflect"
	"testing"

	"github.com/olebedev/emitter"
)

var scraper Scraper

func init() {
	scraper = NewPageScraper(
		client,
		SchemaFromString(`{
			"name": "example title",
			"css": ["body"],
			"properties": [{
				"id":"title",
				"css": ["h1"]
			}]
	 }`))
}
func TestStreamRunner_worker(t *testing.T) {
	type args struct {
		id   int
		urls <-chan string
	}
	tests := []struct {
		name string
		s    *StreamRunner
		args args
	}{
	// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.s.worker(tt.args.id, tt.args.urls)
		})
	}
}

func TestStreamRunner_pool(t *testing.T) {
	tests := []struct {
		name string
		s    *StreamRunner
		want *StreamRunner
	}{
	// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.s.pool(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("StreamRunner.pool() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStreamRunner_Add(t *testing.T) {
	type args struct {
		url string
	}
	tests := []struct {
		name string
		s    *StreamRunner
		args args
		want bool
	}{
	// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.s.Add(tt.args.url); got != tt.want {
				t.Errorf("StreamRunner.Add() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStreamRunner_GetResult(t *testing.T) {
	type args struct {
		url string
	}
	tests := []struct {
		name string
		s    *StreamRunner
		args args
		want <-chan emitter.Event
	}{
		// TODO: Add test cases.
		{
			name: "Parse urls and get results",
			s:    NewStreamRunner(scraper),
			args: args{"http://example.com"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.s.GetResult(tt.args.url); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("StreamRunner.GetResult() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStreamRunner_GetError(t *testing.T) {
	type args struct {
		url string
	}
	tests := []struct {
		name string
		s    *StreamRunner
		args args
		want <-chan emitter.Event
	}{
	// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.s.GetError(tt.args.url); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("StreamRunner.GetError() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStreamRunner_Close(t *testing.T) {
	tests := []struct {
		name string
		s    *StreamRunner
	}{
	// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.s.Close()
		})
	}
}

func TestNewStreamRunner(t *testing.T) {
	type args struct {
		scraper Scraper
	}
	tests := []struct {
		name string
		args args
		want *StreamRunner
	}{
	// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewStreamRunner(tt.args.scraper); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewStreamRunner() = %v, want %v", got, tt.want)
			}
		})
	}
}
