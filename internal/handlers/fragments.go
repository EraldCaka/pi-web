package handlers

import (
	"fmt"
	"html/template"
	"log/slog"
	"sort"
	"strings"

	"github.com/EraldCaka/pi-web/internal/services"
	"github.com/gofiber/fiber/v2"
)

// FragmentsHandler returns small HTML snippets consumed by HTMX polling.
type FragmentsHandler struct {
	device  *services.DeviceService
	users   *services.UserService
	log     *slog.Logger
}

func NewFragmentsHandler(device *services.DeviceService, users *services.UserService, log *slog.Logger) *FragmentsHandler {
	return &FragmentsHandler{device: device, users: users, log: log}
}

func html(c *fiber.Ctx, s string) error {
	c.Set("Content-Type", "text/html; charset=utf-8")
	return c.SendString(s)
}

// Health returns the device health card contents.
func (h *FragmentsHandler) Health(c *fiber.Ctx) error {
	info := h.device.Info()

	deviceName := info.Name
	if deviceName == "" {
		deviceName = "pi"
	}
	mode := info.Mode
	if mode == "" {
		mode = "—"
	}

	online := strings.EqualFold(info.Status, "online")
	dotClass := "dot-red"
	badgeClass := "badge-offline"
	statusText := "Offline"
	if online {
		dotClass = "dot-green"
		badgeClass = "badge-online"
		statusText = "Online"
	}

	return html(c, fmt.Sprintf(`
<div class="card-header">
  <span class="card-title">
    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="2" y="3" width="20" height="14" rx="2"/><path d="M8 21h8M12 17v4"/></svg>
    Device
  </span>
  <span class="badge %s">%s</span>
</div>
<div class="stat-row">
  <div class="stat-item">
    <span class="stat-item-label">Name</span>
    <span class="stat-item-value">%s</span>
  </div>
  <div class="stat-item">
    <span class="stat-item-label">Mode</span>
    <span class="stat-item-value">%s</span>
  </div>
  <div class="stat-item">
    <span class="stat-item-label">Status</span>
    <span class="stat-item-value" style="display:flex;align-items:center;gap:.4rem">
      <span class="dot %s"></span>%s
    </span>
  </div>
</div>`, badgeClass, statusText, template.HTMLEscapeString(deviceName), template.HTMLEscapeString(mode), dotClass, statusText))
}

// Metrics returns the system metrics card contents.
func (h *FragmentsHandler) Metrics(c *fiber.Ctx) error {
	sm, err := h.device.SystemMetrics()
	if err != nil {
		h.log.Warn("system metrics fragment: fetch failed", "error", err)
	}

	uptime := sm.Uptime
	if uptime == "" {
		uptime = "—"
	}

	cpu := sm.CPU
	mem := sm.Memory
	temp := sm.Temp

	return html(c, fmt.Sprintf(`
<div class="card-header">
  <span class="card-title">
    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="22 12 18 12 15 21 9 3 6 12 2 12"/></svg>
    System Metrics
  </span>
  <span style="font-size:.75rem;color:var(--muted)">Uptime: %s</span>
</div>
<div class="grid grid-3">
  <div>
    <div class="progress-wrap">
      <div class="progress-label"><span>CPU</span><span>%.1f%%</span></div>
      <div class="progress-bar"><div class="progress-fill %s" style="width:%.1f%%"></div></div>
    </div>
  </div>
  <div>
    <div class="progress-wrap">
      <div class="progress-label"><span>Memory</span><span>%.1f%%</span></div>
      <div class="progress-bar"><div class="progress-fill %s" style="width:%.1f%%"></div></div>
    </div>
  </div>
  <div>
    <div class="progress-wrap">
      <div class="progress-label"><span>Temp</span><span>%.1f°C</span></div>
      <div class="progress-bar"><div class="progress-fill %s" style="width:%.1f%%"></div></div>
    </div>
  </div>
</div>`,
		template.HTMLEscapeString(uptime),
		cpu, fillClass(cpu), clamp(cpu),
		mem, fillClass(mem), clamp(mem),
		temp, tempFillClass(temp), clamp(temp/100*100),
	))
}

// bmp280NoData is the BMP280 power-on/skip ADC value — not a real measurement.
const bmp280NoData = 524288

// Sensors returns the sensor readings card contents.
func (h *FragmentsHandler) Sensors(c *fiber.Ctx) error {
	data, err := h.device.ReadSensors()
	if err != nil {
		h.log.Warn("sensors fragment: read failed", "error", err)
		return html(c, `
<div class="card-header">
  <span class="card-title">
    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M14 14.76V3.5a2.5 2.5 0 0 0-5 0v11.26a4.5 4.5 0 1 0 5 0z"/></svg>
    Sensors
  </span>
</div>
<div class="empty-state">Device unavailable</div>`)
	}

	if len(data) == 0 {
		return html(c, `
<div class="card-header">
  <span class="card-title">Sensors</span>
</div>
<div class="empty-state">No sensor data</div>`)
	}

	var sb strings.Builder
	sb.WriteString(`<div class="card-header">
  <span class="card-title">
    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M14 14.76V3.5a2.5 2.5 0 0 0-5 0v11.26a4.5 4.5 0 1 0 5 0z"/></svg>
    Sensors
  </span>
</div>`)

	ids := make([]string, 0, len(data))
	for id := range data {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	for _, id := range ids {
		readings := data[id]
		sb.WriteString(fmt.Sprintf(`<div style="margin-bottom:.75rem"><div style="font-size:.78rem;font-weight:600;color:var(--muted);text-transform:uppercase;letter-spacing:.5px;margin-bottom:.5rem">%s</div>`, template.HTMLEscapeString(id)))
		sb.WriteString(`<div class="sensor-grid">`)

		keys := make([]string, 0, len(readings))
		for k := range readings {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		for _, k := range keys {
			if sensorMetaFields[k] {
				continue
			}
			v := readings[k]
			label, value := formatSensorField(k, v)
			sb.WriteString(fmt.Sprintf(`<div class="sensor-reading"><div class="val">%s</div><div class="key">%s</div></div>`,
				template.HTMLEscapeString(value),
				template.HTMLEscapeString(label),
			))
		}
		sb.WriteString(`</div></div>`)
	}

	return html(c, sb.String())
}

// sensorMetaFields are internal I2C / driver fields that add no value to the UI.
var sensorMetaFields = map[string]bool{
	"address": true, "bus": true, "bytes": true,
	"id": true, "type": true, "status": true,
	"hex": true, "length": true,
}

// formatSensorField returns a display label and value for a sensor field.
// temp_raw is converted to °C and pressure_raw is converted to hPa.
func formatSensorField(key string, v any) (label, value string) {
	switch key {
	case "temp_raw":
		raw := int64(toFloat(v))
		if raw == 0 || raw == bmp280NoData {
			return "Temperature", "— °C"
		}
		tc := bmp280TempC(raw)
		return "Temperature", fmt.Sprintf("%.2f °C", tc)

	case "pressure_raw":
		raw := int64(toFloat(v))
		if raw == 0 || raw == bmp280NoData {
			return "Pressure", "— hPa"
		}
		hpa := bmp280PressureHPa(raw)
		return "Pressure", fmt.Sprintf("%.2f hPa", hpa)

	default:
		return key, formatSensorVal(v)
	}
}

// bmp280TempC converts a BMP280 raw temperature ADC value to °C using the
// datasheet compensation formula with typical factory calibration constants.
func bmp280TempC(adcT int64) float64 {
	const digT1, digT2, digT3 = 27504.0, 26435.0, -1000.0
	var1 := (float64(adcT)/16384.0 - digT1/1024.0) * digT2
	var2 := (float64(adcT)/131072.0 - digT1/8192.0) *
		(float64(adcT)/131072.0 - digT1/8192.0) * digT3
	return (var1 + var2) / 5120.0
}

// bmp280PressureHPa converts a BMP280 raw pressure ADC value to hPa.
// It calls bmp280TempC internally for the t_fine value.
func bmp280PressureHPa(adcP int64) float64 {
	// Use a neutral mid-range temperature raw value to derive t_fine
	// when only the pressure value is available.
	const adcTDefault = 519200 // ≈ 25 °C typical room temperature
	const digT1, digT2, digT3 = 27504.0, 26435.0, -1000.0
	const digP1, digP2, digP3 = 36477.0, -10685.0, 3024.0
	const digP4, digP5, digP6 = 2855.0, 140.0, -7.0
	const digP7, digP8, digP9 = 15500.0, -14600.0, 6000.0

	var1T := (float64(adcTDefault)/16384.0 - digT1/1024.0) * digT2
	var2T := (float64(adcTDefault)/131072.0 - digT1/8192.0) *
		(float64(adcTDefault)/131072.0 - digT1/8192.0) * digT3
	tFine := var1T + var2T

	var1P := tFine/2.0 - 64000.0
	var2P := var1P * var1P * digP6 / 32768.0
	var2P += var1P * digP5 * 2.0
	var2P = var2P/4.0 + digP4*65536.0
	var1P = (digP3*var1P*var1P/524288.0 + digP2*var1P) / 524288.0
	var1P = (1.0 + var1P/32768.0) * digP1
	if var1P == 0 {
		return 0
	}
	p := 1048576.0 - float64(adcP)
	p = (p - var2P/4096.0) * 6250.0 / var1P
	p += (digP9*p*p/2147483648.0 + digP8*p + digP7) / 16.0
	return p / 100.0 // Pa → hPa
}

// Users returns the user table for the admin page.
func (h *FragmentsHandler) Users(c *fiber.Ctx) error {
	users, err := h.users.List()
	if err != nil {
		return html(c, `<div class="alert alert-error">Failed to load users</div>`)
	}

	if len(users) == 0 {
		return html(c, `<div class="empty-state">No users found</div>`)
	}

	var sb strings.Builder
	sb.WriteString(`<table><thead><tr>
<th>Email</th><th>Role</th><th>Joined</th><th>Actions</th>
</tr></thead><tbody>`)

	for _, u := range users {
		roleClass := "badge-user"
		if string(u.Role) == "admin" {
			roleClass = "badge-admin"
		}
		sb.WriteString(fmt.Sprintf(`<tr id="user-%s">
  <td>%s</td>
  <td><span class="badge %s">%s</span></td>
  <td style="color:var(--muted);font-size:.8rem">%s</td>
  <td>
    <div style="display:flex;gap:.4rem">
      %s
      <button class="btn btn-sm btn-red"
              hx-delete="/users/%s"
              hx-confirm="Delete this user?"
              hx-target="#user-%s"
              hx-swap="outerHTML">Delete</button>
    </div>
  </td>
</tr>`,
			u.ID,
			template.HTMLEscapeString(u.Email),
			roleClass, string(u.Role),
			u.CreatedAt.Format("Jan 2, 2006"),
			func() string {
				if string(u.Role) != "admin" {
					return fmt.Sprintf(`<button class="btn btn-sm btn-orange"
              hx-post="/users/%s/promote"
              hx-target="#users-table"
              hx-get="/fragments/users"
              hx-swap="innerHTML"
              hx-trigger="click">Promote</button>`, u.ID)
				}
				return ""
			}(),
			u.ID, u.ID,
		))
	}
	sb.WriteString(`</tbody></table>`)
	return html(c, sb.String())
}

// ── helpers ──────────────────────────────────────────────────────────────────

func toFloat(v any) float64 {
	if v == nil {
		return 0
	}
	switch n := v.(type) {
	case float64:
		return n
	case float32:
		return float64(n)
	case int:
		return float64(n)
	}
	return 0
}

func clamp(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 100 {
		return 100
	}
	return v
}

func fillClass(pct float64) string {
	switch {
	case pct >= 80:
		return "fill-red"
	case pct >= 60:
		return "fill-orange"
	default:
		return "fill-blue"
	}
}

func tempFillClass(t float64) string {
	switch {
	case t >= 80:
		return "fill-red"
	case t >= 60:
		return "fill-orange"
	default:
		return "fill-green"
	}
}

func formatSensorVal(v any) string {
	if v == nil {
		return "—"
	}
	switch n := v.(type) {
	case float64:
		return fmt.Sprintf("%.2f", n)
	case float32:
		return fmt.Sprintf("%.2f", n)
	case int, int64:
		return fmt.Sprintf("%d", n)
	default:
		return fmt.Sprintf("%v", n)
	}
}
