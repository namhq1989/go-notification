package notification

import "testing"

func TestPlatform_IsValid(t *testing.T) {
	tests := []struct {
		platform Platform
		expected bool
	}{
		{PlatformIOS, true},
		{PlatformAndroid, true},
		{PlatformUnknown, false},
		{Platform("web"), false},
		{Platform(""), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.platform), func(t *testing.T) {
			if tt.platform.IsValid() != tt.expected {
				t.Errorf("Platform(%q).IsValid() = %v, want %v", tt.platform, tt.platform.IsValid(), tt.expected)
			}
		})
	}
}

func TestToPlatform(t *testing.T) {
	tests := []struct {
		input    string
		expected Platform
	}{
		{"ios", PlatformIOS},
		{"android", PlatformAndroid},
		{"", PlatformUnknown},
		{"web", PlatformUnknown},
		{"IOS", PlatformUnknown}, // case sensitive
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := ToPlatform(tt.input)
			if result != tt.expected {
				t.Errorf("ToPlatform(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestPlatform_String(t *testing.T) {
	if PlatformIOS.String() != "ios" {
		t.Error("PlatformIOS.String() should be 'ios'")
	}
	if PlatformAndroid.String() != "android" {
		t.Error("PlatformAndroid.String() should be 'android'")
	}
}
