package main

import (
	"fmt"

	"github.com/uber/h3-go/v4"
)

func GeoJsonToH3GeoPolygon(geojson GeoJSONGeometry) (h3.GeoPolygon, error) {
	if geojson.Type != "Polygon" {
		return h3.GeoPolygon{}, fmt.Errorf("unsupported GeoJSON type: %s", geojson.Type)
	}

	coords, ok := geojson.Coordinates.([]interface{})
	if !ok || len(coords) == 0 {
		return h3.GeoPolygon{}, fmt.Errorf("invalid geoJson coords")
	}

	var geoPolygon h3.GeoPolygon

	// Outer boundary
	outerRing, ok := coords[0].([]interface{})
	if !ok {
		return h3.GeoPolygon{}, fmt.Errorf("invalid outer ring coords")
	}

	var geoLoop h3.GeoLoop
	for _, coord := range outerRing {
		point, ok := coord.([]interface{})
		if !ok || len(point) != 2 {
			return h3.GeoPolygon{}, fmt.Errorf("invalid point in outer ring")
		}
		lat, latOk := point[1].(float64)
		lng, lngOk := point[0].(float64)
		if !latOk || !lngOk {
			return h3.GeoPolygon{}, fmt.Errorf("invalid point coords")
		}

		geoLoop = append(geoLoop, h3.LatLng{Lat: lat, Lng: lng})
	}
	geoPolygon.GeoLoop = geoLoop

	// TODO: Extract holes (inner rings)
	return geoPolygon, nil
}
