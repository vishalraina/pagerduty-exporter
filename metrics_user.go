package main

import (
	"context"
	"github.com/PagerDuty/go-pagerduty"
	"github.com/prometheus/client_golang/prometheus"
	prometheusCommon "github.com/webdevops/go-prometheus-common"
	"strings"
)

type MetricsCollectorUser struct {
	CollectorProcessorGeneral

	prometheus struct {
		user *prometheus.GaugeVec
	}

	teamListOpt []string
}

func (m *MetricsCollectorUser) Setup(collector *CollectorGeneral) {
	m.CollectorReference = collector

	m.prometheus.user = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "pagerduty_user_info",
			Help: "PagerDuty user",
		},
		[]string{
			"userID",
			"userName",
			"userMail",
			"userAvatar",
			"userColor",
			"userJobTitle",
			"userRole",
			"userTimezone",
			"userTeam",
		},
	)

	prometheus.MustRegister(m.prometheus.user)
}

func (m *MetricsCollectorUser) Reset() {
	m.prometheus.user.Reset()
}

func (m *MetricsCollectorUser) Collect(ctx context.Context, callback chan<- func()) {
	listOpts := pagerduty.ListUsersOptions{}
	listOpts.Limit = PagerdutyListLimit
	listOpts.Offset = 0

	if len(m.teamListOpt) > 0 {
		listOpts.TeamIDs = m.teamListOpt
	}

	userMetricList := prometheusCommon.NewMetricsList()

	for {
		m.logger().Debugf("fetch users (offset: %v, limit:%v)", listOpts.Offset, listOpts.Limit)

		list, err := PagerDutyClient.ListUsers(listOpts)
		m.CollectorReference.PrometheusAPICounter().WithLabelValues("ListUsers").Inc()

		if err != nil {
			m.logger().Panic(err)
		}

		for _, user := range list.Users {
			user_teams := make([]string, 0)
			//summaries := make([]string, 0)
			for _, team := range user.Teams {
				user_teams = append(user_teams, team.APIObject.Summary)
			}

			userMetricList.AddInfo(prometheus.Labels{
				"userID":       user.ID,
				"userName":     user.Name,
				"userMail":     user.Email,
				"userAvatar":   user.AvatarURL,
				"userColor":    user.Color,
				"userJobTitle": user.JobTitle,
				"userRole":     user.Role,
				"userTimezone": user.Timezone,
				"userTeam":     strings.Join(user_teams, ","),
			})
		}

		listOpts.Offset += list.Limit
		if !list.More {
			break
		}
	}

	// set metrics
	callback <- func() {
		userMetricList.GaugeSet(m.prometheus.user)
	}
}
