package main

type sToken struct {
	Token string `json:"token"`
}

type GeoQueryClause struct {
	Lat      float64 `json:"lat"`
	Lng      float64 `json:"lng"`
	Unit     string  `json:"unit"`
	Distance int     `json:"distance"`
}

type Response struct {
	Data SearchJobCard `json:"data"`
}

type SearchJobCard struct {
	JobCard struct {
		NextToken any    `json:"nextToken"`
		Cards     []Card `json:"jobCards"`
		TypeName  string `json:"__typename"`
	} `json:"searchJobCardsByLocation"`
}

type Card struct {
	JobId    string `json:"jobId"`
	JobTitle string `json:"jobTitle"`
	City     string `json:"city"`
	State    string `json:"state"`
}
