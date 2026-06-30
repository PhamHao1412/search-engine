package service

import (
	"reflect"
	"testing"
)

func TestSearchService_ExpandQuery(t *testing.T) {
	s := &searchService{}

	synonyms := map[string][]string{
		"keyboard":        {"ban phim", "ban phim co"},
		"chuot gaming":    {"chuot choi game"},
		"chuot choi game": {"chuot gaming"},
		"tai nghe":        {"headset", "headphone"},
	}

	tests := []struct {
		name     string
		query    string
		expected [][]string
	}{
		{
			name:  "No synonyms match",
			query: "chuột không dây logitech",
			expected: [][]string{
				{"chuột"},
				{"không"},
				{"dây"},
				{"logitech"},
			},
		},
		{
			name:  "Single word synonym match",
			query: "bàn phím cơ keyboard",
			expected: [][]string{
				{"bàn"},
				{"phím"},
				{"cơ"},
				{"keyboard", "ban phim", "ban phim co"},
			},
		},
		{
			name:  "Phrase synonym match with diacritics",
			query: "mua chuột gaming giá rẻ",
			expected: [][]string{
				{"mua"},
				{"chuột gaming", "chuot choi game"},
				{"giá"},
				{"rẻ"},
			},
		},
		{
			name:  "Greedy phrase matching longest matches first",
			query: "tai nghe gaming",
			expected: [][]string{
				{"tai nghe", "headset", "headphone"},
				{"gaming"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := s.ExpandQuery(tt.query, synonyms)
			if !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("ExpandQuery() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestRemoveDiacritics(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Chuột Gaming", "chuot gaming"},
		{"Bàn Phím Cơ CựC ĐẹP", "ban phim co cuc dep"},
		{"tai nghe không dây", "tai nghe khong day"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := removeDiacritics(tt.input)
			if got != tt.expected {
				t.Errorf("removeDiacritics() = %q, want %q", got, tt.expected)
			}
		})
	}
}
