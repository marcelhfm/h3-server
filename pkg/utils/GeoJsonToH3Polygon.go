package utils

import (
	"fmt"

	"github.com/marcelhfm/h3-server/pkg/types"
	"github.com/uber/h3-go/v4"
)

func GeoJsonToH3GeoPolygon(geojson typings.GeoJSONGeometry) (h3.GeoPolygon, error) {
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

	// Extract holes (inner rings)
	for i := 1; i < len(coords); i++ {
		holeRing, ok := coords[i].([]interface{})
		if !ok {
			return h3.GeoPolygon{}, fmt.Errorf("Invalid hole ring coords")
		}

		var holeLoop h3.GeoLoop
		for _, coord := range holeRing {
			point, ok := coord.([]interface{})
			if !ok || len(point) != 2 {
				return h3.GeoPolygon{}, fmt.Errorf("Invalid point in hole ring")
			}
			lat, latOk := point[1].(float64)
			lng, lngOk := point[0].(float64)
			if !latOk || !lngOk {
				return h3.GeoPolygon{}, fmt.Errorf("invalid point coords")
			}
			holeLoop = append(holeLoop, h3.LatLng{Lat: lat, Lng: lng})
		}
		geoPolygon.Holes = append(geoPolygon.Holes, holeLoop)
	}

	return geoPolygon, nil
}
