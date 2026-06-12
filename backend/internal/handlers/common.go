package handlers

import (
	"context"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"stone-relic-monitor/internal/config"
	"stone-relic-monitor/internal/db"
)

type CommonHandler struct {
	cfg *config.Config
	db  *db.ClickHouse
}

func NewCommonHandler(cfg *config.Config, db *db.ClickHouse) *CommonHandler {
	return &CommonHandler{cfg: cfg, db: db}
}

func (h *CommonHandler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"version": "2.0.0",
		"modules": []string{"ethercat_ingest", "scaling_model", "laser_threshold", "alert_ws"},
	})
}

func (h *CommonHandler) ListRelics(c *gin.Context) {
	ctx := context.Background()
	query := `SELECT id, name, location, material, construction_year, dimensions, status
		FROM stone_relic ORDER BY id`

	rows, err := h.db.Query(ctx, query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var relics []map[string]interface{}
	for rows.Next() {
		var id uint64
		var name, location, material string
		var constructionYear int32
		var dimensions, status string
		if err := rows.Scan(&id, &name, &location, &material, &constructionYear, &dimensions, &status); err == nil {
			relics = append(relics, map[string]interface{}{
				"id":                id,
				"name":              name,
				"location":          location,
				"material":          material,
				"construction_year": constructionYear,
				"dimensions":        dimensions,
				"status":            status,
			})
		}
	}
	c.JSON(http.StatusOK, relics)
}

func (h *CommonHandler) GetRelic(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	ctx := context.Background()
	query := `SELECT id, name, location, material, construction_year, dimensions, status, description
		FROM stone_relic WHERE id = ?`

	rows, err := h.db.Query(ctx, query, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	if rows.Next() {
		var relic struct {
			ID               uint64  `json:"id"`
			Name             string  `json:"name"`
			Location         string  `json:"location"`
			Material         string  `json:"material"`
			ConstructionYear int32   `json:"construction_year"`
			Dimensions       string  `json:"dimensions"`
			Status           string  `json:"status"`
			Description      string  `json:"description"`
		}
		if err := rows.Scan(&relic.ID, &relic.Name, &relic.Location, &relic.Material,
			&relic.ConstructionYear, &relic.Dimensions, &relic.Status, &relic.Description); err == nil {
			c.JSON(http.StatusOK, relic)
			return
		}
	}
	c.JSON(http.StatusNotFound, gin.H{"error": "relic not found"})
}

func (h *CommonHandler) ListSensors(c *gin.Context) {
	relicID, err := strconv.ParseUint(c.Query("relic_id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid relic_id"})
		return
	}

	ctx := context.Background()
	query := `SELECT id, relic_id, type, model, unit, installation_date, status, location_x, location_y, location_z
		FROM sensor WHERE relic_id = ? ORDER BY type, id`

	rows, err := h.db.Query(ctx, query, relicID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var sensors []map[string]interface{}
	for rows.Next() {
		var id, relicId uint64
		var sensorType, model, unit, installationDate, status string
		var locX, locY, locZ float64
		if err := rows.Scan(&id, &relicId, &sensorType, &model, &unit, &installationDate, &status, &locX, &locY, &locZ); err == nil {
			sensors = append(sensors, map[string]interface{}{
				"id":                id,
				"relic_id":          relicId,
				"type":              sensorType,
				"model":             model,
				"unit":              unit,
				"installation_date": installationDate,
				"status":            status,
				"location_x":        locX,
				"location_y":        locY,
				"location_z":        locZ,
			})
		}
	}
	c.JSON(http.StatusOK, sensors)
}

func (h *CommonHandler) GetSensor(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	ctx := context.Background()
	query := `SELECT id, relic_id, type, model, unit, installation_date, status, location_x, location_y, location_z, description
		FROM sensor WHERE id = ?`

	rows, err := h.db.Query(ctx, query, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	if rows.Next() {
		var sensor struct {
			ID             uint64  `json:"id"`
			RelicID        uint64  `json:"relic_id"`
			Type           string  `json:"type"`
			Model          string  `json:"model"`
			Unit           string  `json:"unit"`
			InstallationDate string `json:"installation_date"`
			Status         string  `json:"status"`
			LocationX      float64 `json:"location_x"`
			LocationY      float64 `json:"location_y"`
			LocationZ      float64 `json:"location_z"`
			Description    string  `json:"description"`
		}
		if err := rows.Scan(&sensor.ID, &sensor.RelicID, &sensor.Type, &sensor.Model,
			&sensor.Unit, &sensor.InstallationDate, &sensor.Status,
			&sensor.LocationX, &sensor.LocationY, &sensor.LocationZ, &sensor.Description); err == nil {
			c.JSON(http.StatusOK, sensor)
			return
		}
	}
	c.JSON(http.StatusNotFound, gin.H{"error": "sensor not found"})
}

func (h *CommonHandler) RegisterRoutes(r *gin.RouterGroup) {
	relics := r.Group("/relics")
	{
		relics.GET("", h.ListRelics)
		relics.GET("/:id", h.GetRelic)
	}

	sensors := r.Group("/sensors")
	{
		sensors.GET("", h.ListSensors)
		sensors.GET("/:id", h.GetSensor)
	}
}
