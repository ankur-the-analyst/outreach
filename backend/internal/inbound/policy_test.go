package inbound

import "testing"

func TestPolicyEnrolment(t *testing.T) {
	t.Parallel()
	cases := []struct {
		class Class
		want  string
	}{
		{ClassInterested, "replied"},
		{ClassInterview, "replied"},
		{ClassQuestion, "replied"},
		{ClassReferral, "replied"},
		{ClassOther, "replied"},
		{ClassNotNow, "stopped"},
		{ClassRejection, "stopped"},
		{ClassUnsubscribe, "unsubscribed"},
		{ClassComplaint, "unsubscribed"},
		{ClassDSN, "bounced"},
		{ClassOutOfOffice, "held"},
	}
	for _, tc := range cases {
		if got := Policy(tc.class).Enrolment; got != tc.want {
			t.Fatalf("%s: %s want %s", tc.class, got, tc.want)
		}
	}
}
