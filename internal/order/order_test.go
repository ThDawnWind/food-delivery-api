package order

import "testing"

func TestCanTransitionStatus(t *testing.T) {
	tests := []struct {
		name string
		from Status
		to   Status
		want bool
	}{
		{
			name: "new to confirmed",
			from: StatusNew,
			to:   StatusConfirmed,
			want: true,
		},
		{
			name: "new to cancelled",
			from: StatusNew,
			to:   StatusCancelled,
			want: true,
		},
		{
			name: "confirmed to cooking",
			from: StatusConfirmed,
			to:   StatusCooking,
			want: true,
		},
		{
			name: "cooking to ready",
			from: StatusCooking,
			to:   StatusReady,
			want: true,
		},
		{
			name: "ready to delivering",
			from: StatusReady,
			to:   StatusDelivering,
			want: true,
		},
		{
			name: "delivering to completed",
			from: StatusDelivering,
			to:   StatusCompleted,
			want: true,
		},

		{
			name: "new to completed",
			from: StatusNew,
			to:   StatusCompleted,
			want: false,
		},
		{
			name: "completed to cooking",
			from: StatusCompleted,
			to:   StatusCooking,
			want: false,
		},
		{
			name: "cancelled to confirmed",
			from: StatusCancelled,
			to:   StatusConfirmed,
			want: false,
		},
		{
			name: "delivering to cancelled",
			from: StatusDelivering,
			to:   StatusCancelled,
			want: false,
		},
		{
			name: "same status",
			from: StatusCooking,
			to:   StatusCooking,
			want: false,
		},
		{
			name: "unknown status",
			from: Status("unknown"),
			to:   StatusConfirmed,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CanTransitionStatus(
				tt.from,
				tt.to,
			)

			if got != tt.want {
				t.Errorf(
					"CanTransitionStatus(%q, %q) = %v, want %v",
					tt.from,
					tt.to,
					got,
					tt.want,
				)
			}
		})
	}
}
