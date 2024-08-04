package main

import (
	"encoding/json"
	"io"
	"net/http"

	l "github.com/marcelhfm/h3-server/pkg/log"
	typings "github.com/marcelhfm/h3-server/pkg/types"
	"github.com/marcelhfm/h3-server/pkg/utils"
	"github.com/uber/h3-go/v4"
)

type CreateH3IndexRequest struct {
	Geometries []typings.GeoJSONGeometry `json:"geometries"`
	Resolution int                       `json:"resolution"`
	Compact    bool                      `json:"compact"`
}

type GeometryH3Obj struct {
	Geometry  typings.GeoJSONGeometry `json:"geometry"`
	H3Indices []h3.Cell               `json:"h3_indices"`
}

type CreateH3IndexResponse struct {
	Result []GeometryH3Obj `json:"result"`
}

func CreateIndexHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			l.Log.Error().Msgf("Failed to read request body %v", r.Body)
			http.Error(w, "Failed to read request body", http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		var h3Request CreateH3IndexRequest
		err = json.Unmarshal(body, &h3Request)
		if err != nil {
			l.Log.Error().Msgf("Invalid JSON format for body: %v", r.Body)
			http.Error(w, "Invalid JSON format", http.StatusBadRequest)
			return
		}

		if h3Request.Resolution == 0 {
			l.Log.Error().Msg("No resolution provided")
			http.Error(w, "No resulution provided", http.StatusBadRequest)
			return
		}

		l.Log.Debug().Msgf("Creating h3 cells for %d geometries with resolution %d. Compacting: %t", len(h3Request.Geometries), h3Request.Resolution, h3Request.Compact)

		var results []GeometryH3Obj
		for _, geometry := range h3Request.Geometries {
			var cells []h3.Cell
			if geometry.Type == "Point" {
				coords := geometry.Coordinates.([]interface{})
				lng := coords[0].(float64)
				lat := coords[1].(float64)

				latLng := h3.NewLatLng(lat, lng)
				cell := h3.LatLngToCell(latLng, h3Request.Resolution)

				cells = append(cells, cell)
			} else if geometry.Type == "Polygon" {
				geoPolygon, err := utils.GeoJsonToH3GeoPolygon(geometry)
				if err != nil {
					l.Log.Error().Msgf("Error while creating geopolygon: %v", err)
					http.Error(w, "Error while creating geopolygon", http.StatusInternalServerError)
					return
				}

				polyCells := h3.PolygonToCells(geoPolygon, h3Request.Resolution)
				cells = append(cells, polyCells...)
			} else if geometry.Type == "MultiPolygon" {
				multiPolygonCoords, ok := geometry.Coordinates.([]interface{})
				if !ok {
					l.Log.Error().Msg("Invalid MultiPolygon coordinates format")
					http.Error(w, "Invalid MultiPolygon coordinates format", http.StatusBadRequest)
					return
				}

				for _, polygonCoords := range multiPolygonCoords {
					polygonGeom := typings.GeoJSONGeometry{
						Type:        "Polygon",
						Coordinates: polygonCoords,
					}

					geoPolygon, err := utils.GeoJsonToH3GeoPolygon(polygonGeom)
					if err != nil {
						l.Log.Error().Msgf("Error while creating geopolygon: %v", err)
						http.Error(w, "Error while creating geopolygon", http.StatusInternalServerError)
						return
					}

					polyCells := h3.PolygonToCells(geoPolygon, h3Request.Resolution)
					cells = append(cells, polyCells...)
				}
			}

			results = append(results, GeometryH3Obj{Geometry: geometry, H3Indices: cells})
		}

		if h3Request.Compact {
			for _, result := range results {
				if len(result.H3Indices) == 0 {
					l.Log.Warn().Msg("No h3 indices to compact")
					continue
				}
				uncompacted := result.H3Indices
				compacted := h3.CompactCells(uncompacted)

				result.H3Indices = compacted
			}
		}

		response := CreateH3IndexResponse{Result: results}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}
