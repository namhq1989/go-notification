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
		{"IOS", PlatformIOS},         // case insensitive
		{"Android", PlatformAndroid}, // case insensitive
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

func TestPriority_IsValid(t *testing.T) {
	tests := []struct {
		priority Priority
		expected bool
	}{
		{PriorityNormal, true},
		{PriorityHigh, true},
		{PriorityUnknown, false},
		{Priority(""), false},
		{Priority("critical"), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.priority), func(t *testing.T) {
			if tt.priority.IsValid() != tt.expected {
				t.Errorf("Priority(%q).IsValid() = %v, want %v", tt.priority, tt.priority.IsValid(), tt.expected)
			}
		})
	}
}

func TestToPriority(t *testing.T) {
	tests := []struct {
		input    string
		expected Priority
	}{
		{"high", PriorityHigh},
		{"normal", PriorityNormal},
		{"HIGH", PriorityHigh},     // case insensitive
		{"Normal", PriorityNormal}, // case insensitive
		{"High", PriorityHigh},     // case insensitive
		{"NORMAL", PriorityNormal}, // case insensitive
		{"", PriorityUnknown},
		{"critical", PriorityUnknown},
		{"low", PriorityUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := ToPriority(tt.input)
			if result != tt.expected {
				t.Errorf("ToPriority(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}
