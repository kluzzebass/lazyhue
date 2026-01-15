package hue

import (
	"testing"

	"github.com/kluzzebass/lazyhue/internal/hueclient"
)

// Helper to create a test light
func makeTestLight(on bool, brightness float32) hueclient.LightGet {
	onVal := on
	brightnessVal := hueclient.Brightness(brightness)
	return hueclient.LightGet{
		On: &hueclient.On{On: &onVal},
		Dimming: &struct {
			Brightness  *hueclient.Brightness `json:"brightness,omitempty"`
			MinDimLevel *float32              `json:"min_dim_level,omitempty"`
		}{Brightness: &brightnessVal},
	}
}

// Helper to create a test light that's off
func makeOffLight() hueclient.LightGet {
	offVal := false
	return hueclient.LightGet{
		On: &hueclient.On{On: &offVal},
	}
}

// Helper to create a test light with no dimming info
func makeNoDimmingLight() hueclient.LightGet {
	onVal := true
	return hueclient.LightGet{
		On: &hueclient.On{On: &onVal},
	}
}

func TestCalculateRoomAggregate_EmptyList(t *testing.T) {
	brightness, color := CalculateRoomAggregate(nil)
	if brightness != 0 {
		t.Errorf("Expected brightness 0 for nil list, got %f", brightness)
	}
	if color != "" {
		t.Errorf("Expected empty color for nil list, got %q", color)
	}

	brightness, color = CalculateRoomAggregate([]hueclient.LightGet{})
	if brightness != 0 {
		t.Errorf("Expected brightness 0 for empty slice, got %f", brightness)
	}
	if color != "" {
		t.Errorf("Expected empty color for empty slice, got %q", color)
	}
}

func TestCalculateRoomAggregate_AllOff(t *testing.T) {
	lights := []hueclient.LightGet{
		makeOffLight(),
		makeOffLight(),
	}
	brightness, color := CalculateRoomAggregate(lights)
	if brightness != 0 {
		t.Errorf("Expected brightness 0 when all lights off, got %f", brightness)
	}
	if color != "" {
		t.Errorf("Expected empty color when all lights off, got %q", color)
	}
}

func TestCalculateRoomAggregate_SingleLightOn(t *testing.T) {
	lights := []hueclient.LightGet{
		makeTestLight(true, 75.0),
	}
	brightness, color := CalculateRoomAggregate(lights)
	if brightness != 75.0 {
		t.Errorf("Expected brightness 75, got %f", brightness)
	}
	// Color should be non-empty (defaults to warm white for non-color lights)
	if color == "" {
		t.Errorf("Expected non-empty color")
	}
}

func TestCalculateRoomAggregate_MultipleLightsAverage(t *testing.T) {
	lights := []hueclient.LightGet{
		makeTestLight(true, 50.0),
		makeTestLight(true, 100.0),
	}
	brightness, _ := CalculateRoomAggregate(lights)
	expected := 75.0 // (50 + 100) / 2
	if brightness != expected {
		t.Errorf("Expected brightness %f, got %f", expected, brightness)
	}
}

func TestCalculateRoomAggregate_MixedOnOff(t *testing.T) {
	lights := []hueclient.LightGet{
		makeTestLight(true, 80.0),
		makeTestLight(false, 100.0), // Should be ignored (off)
		makeTestLight(true, 40.0),
	}
	brightness, _ := CalculateRoomAggregate(lights)
	expected := 60.0 // (80 + 40) / 2, ignoring the off light
	if brightness != expected {
		t.Errorf("Expected brightness %f, got %f", expected, brightness)
	}
}

func TestCalculateRoomAggregate_NoDimmingDefaults100(t *testing.T) {
	lights := []hueclient.LightGet{
		makeNoDimmingLight(),
	}
	brightness, _ := CalculateRoomAggregate(lights)
	if brightness != 100.0 {
		t.Errorf("Expected brightness 100 when no dimming info, got %f", brightness)
	}
}

func TestCalculateRoomAggregate_ColorFormat(t *testing.T) {
	lights := []hueclient.LightGet{
		makeTestLight(true, 100.0),
	}
	_, color := CalculateRoomAggregate(lights)
	// Color should be a valid hex color
	if len(color) != 7 || color[0] != '#' {
		t.Errorf("Expected color in #rrggbb format, got %q", color)
	}
}
