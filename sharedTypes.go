package main

type GeoJSONFeature struct {
	Type       string          `json:"type"`
	Properties interface{}     `json:"properties"`
	Geometry   GeoJSONGeometry `json:"geometry"`
}

type GeoJSONGeometry struct {
	Type        string      `json:"type"`
	Coordinates interface{} `json:"coordinates"`
}
