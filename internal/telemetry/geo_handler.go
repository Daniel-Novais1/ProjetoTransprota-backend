package telemetry

import (
	"fmt"
	"net/http"
	"time"

	"github.com/Daniel-Novais1/ProjetoTransprota-backend/internal/logger"
	"github.com/gin-gonic/gin"
)

type GeoController struct {
	repo *Repository
}

func NewGeoController(repo *Repository) *GeoController {
	return &GeoController{repo: repo}
}

func (c *GeoController) ReceiveBusUpdate(ctx *gin.Context) {
	start := time.Now()

	var update BusUpdate
	if err := ctx.ShouldBindJSON(&update); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := c.repo.AddBusLocation(ctx.Request.Context(), &update); err != nil {
		logger.Error("GEO", "Failed to add bus location: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add bus location"})
		return
	}

	ctx.JSON(http.StatusAccepted, gin.H{
		"msg": "Bus update received",
		"pms": time.Since(start).Milliseconds(),
	})
}

func (c *GeoController) GetBusLocations(ctx *gin.Context) {
	start := time.Now()

	lat := ctx.Query("lat")
	lng := ctx.Query("lng")
	radius := ctx.DefaultQuery("radius", "1000")
	route := ctx.Query("route")

	if lat == "" || lng == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "lat and lng required"})
		return
	}

	query := &GeoRadiusQuery{
		CenterLat:    parseLat(lat),
		CenterLng:    parseLng(lng),
		RadiusMeters: parseRadius(radius),
		RouteFilter:  route,
	}

	response, err := c.repo.GetBusLocationsByRadius(ctx.Request.Context(), query)
	if err != nil {
		logger.Error("GEO", "Failed to get bus locations: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get bus locations"})
		return
	}

	logger.Info("GEO", "Enviando %d ônibus para o Frontend", response.Count)
	ctx.JSON(http.StatusOK, gin.H{
		"cnt":   response.Count,
		"buses": response.Buses,
		"pms":   time.Since(start).Milliseconds(),
	})
}

func (c *GeoController) GetAllBuses(ctx *gin.Context) {
	start := time.Now()

	buses, err := c.repo.GetAllActiveBuses(ctx.Request.Context())
	if err != nil {
		logger.Error("GEO", "Failed to get all buses: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get all buses"})
		return
	}

	logger.Info("GEO", "Enviando %d ônibus para o Frontend (GetAllBuses)", len(buses))
	ctx.JSON(http.StatusOK, gin.H{
		"cnt":   len(buses),
		"buses": buses,
		"pms":   time.Since(start).Milliseconds(),
	})
}

func parseLat(s string) float64 {
	var f float64
	fmt.Sscanf(s, "%f", &f)
	return f
}

func parseLng(s string) float64 {
	var f float64
	fmt.Sscanf(s, "%f", &f)
	return f
}

func parseRadius(s string) int {
	var i int
	fmt.Sscanf(s, "%d", &i)
	return i
}
