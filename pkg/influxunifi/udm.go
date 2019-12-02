package influxunifi

import (
	"golift.io/unifi"
)

// Combines concatenates N maps. This will delete things if not used with caution.
func Combine(in ...map[string]interface{}) map[string]interface{} {
	out := make(map[string]interface{})
	for i := range in {
		for k := range in[i] {
			out[k] = in[i][k]
		}
	}
	return out
}

// batchSysStats is used by all device types.
func (u *InfluxUnifi) batchSysStats(s unifi.SysStats, ss unifi.SystemStats) map[string]interface{} {
	return map[string]interface{}{
		"loadavg_1":     s.Loadavg1.Val,
		"loadavg_5":     s.Loadavg5.Val,
		"loadavg_15":    s.Loadavg15.Val,
		"mem_used":      s.MemUsed.Val,
		"mem_buffer":    s.MemBuffer.Val,
		"mem_total":     s.MemTotal.Val,
		"cpu":           ss.CPU.Val,
		"mem":           ss.Mem.Val,
		"system_uptime": ss.Uptime.Val,
	}
}

// batchUDM generates Unifi Gateway datapoints for InfluxDB.
// These points can be passed directly to influx.
func (u *InfluxUnifi) batchUDM(r report, s *unifi.UDM) {
	if s.Stat.Sw == nil {
		s.Stat.Sw = &unifi.Sw{}
	}
	if s.Stat.Gw == nil {
		s.Stat.Gw = &unifi.Gw{}
	}
	tags := map[string]string{
		"mac":       s.Mac,
		"site_name": s.SiteName,
		"name":      s.Name,
		"version":   s.Version,
		"model":     s.Model,
		"serial":    s.Serial,
		"type":      s.Type,
	}
	fields := Combine(map[string]interface{}{
		"ip":                             s.IP,
		"bytes":                          s.Bytes.Val,
		"last_seen":                      s.LastSeen.Val,
		"license_state":                  s.LicenseState,
		"guest-num_sta":                  s.GuestNumSta.Val,
		"rx_bytes":                       s.RxBytes.Val,
		"tx_bytes":                       s.TxBytes.Val,
		"uptime":                         s.Uptime.Val,
		"state":                          s.State.Val,
		"user-num_sta":                   s.UserNumSta.Val,
		"version":                        s.Version,
		"num_desktop":                    s.NumDesktop.Val,
		"num_handheld":                   s.NumHandheld.Val,
		"num_mobile":                     s.NumMobile.Val,
		"speedtest-status_latency":       s.SpeedtestStatus.Latency.Val,
		"speedtest-status_runtime":       s.SpeedtestStatus.Runtime.Val,
		"speedtest-status_ping":          s.SpeedtestStatus.StatusPing.Val,
		"speedtest-status_xput_download": s.SpeedtestStatus.XputDownload.Val,
		"speedtest-status_xput_upload":   s.SpeedtestStatus.XputUpload.Val,
		"lan-rx_bytes":                   s.Stat.Gw.LanRxBytes.Val,
		"lan-rx_packets":                 s.Stat.Gw.LanRxPackets.Val,
		"lan-tx_bytes":                   s.Stat.Gw.LanTxBytes.Val,
		"lan-tx_packets":                 s.Stat.Gw.LanTxPackets.Val,
	}, u.batchSysStats(s.SysStats, s.SystemStats))
	r.send(&metric{Table: "usg", Tags: tags, Fields: fields})
	u.batchNetTable(r, tags, s.NetworkTable)
	u.batchUSGwans(r, tags, s.Wan1, s.Wan2)

	tags = map[string]string{
		"mac":       s.Mac,
		"site_name": s.SiteName,
		"name":      s.Name,
		"version":   s.Version,
		"model":     s.Model,
		"serial":    s.Serial,
		"type":      s.Type,
	}
	fields = map[string]interface{}{
		"guest-num_sta":   s.GuestNumSta.Val,
		"ip":              s.IP,
		"bytes":           s.Bytes.Val,
		"last_seen":       s.LastSeen.Val,
		"rx_bytes":        s.RxBytes.Val,
		"tx_bytes":        s.TxBytes.Val,
		"uptime":          s.Uptime.Val,
		"state":           s.State.Val,
		"stat_bytes":      s.Stat.Sw.Bytes.Val,
		"stat_rx_bytes":   s.Stat.Sw.RxBytes.Val,
		"stat_rx_crypts":  s.Stat.Sw.RxCrypts.Val,
		"stat_rx_dropped": s.Stat.Sw.RxDropped.Val,
		"stat_rx_errors":  s.Stat.Sw.RxErrors.Val,
		"stat_rx_frags":   s.Stat.Sw.RxFrags.Val,
		"stat_rx_packets": s.Stat.Sw.TxPackets.Val,
		"stat_tx_bytes":   s.Stat.Sw.TxBytes.Val,
		"stat_tx_dropped": s.Stat.Sw.TxDropped.Val,
		"stat_tx_errors":  s.Stat.Sw.TxErrors.Val,
		"stat_tx_packets": s.Stat.Sw.TxPackets.Val,
		"stat_tx_retries": s.Stat.Sw.TxRetries.Val,
	}
	r.send(&metric{Table: "usw", Tags: tags, Fields: fields})
	u.batchPortTable(r, tags, s.PortTable)

	if s.Stat.Ap == nil {
		return
		// we're done now. the following code process UDM (non-pro) UAP data.
	}
	tags = map[string]string{
		"mac":       s.Mac,
		"site_name": s.SiteName,
		"name":      s.Name,
		"version":   s.Version,
		"model":     s.Model,
		"serial":    s.Serial,
		"type":      s.Type,
	}
	fields = u.processUAPstats(s.Stat.Ap)
	fields["ip"] = s.IP
	fields["bytes"] = s.Bytes.Val
	fields["last_seen"] = s.LastSeen.Val
	fields["rx_bytes"] = s.RxBytes.Val
	fields["tx_bytes"] = s.TxBytes.Val
	fields["uptime"] = s.Uptime.Val
	fields["state"] = s.State
	fields["user-num_sta"] = int(s.UserNumSta.Val)
	fields["guest-num_sta"] = int(s.GuestNumSta.Val)
	fields["num_sta"] = s.NumSta.Val
	r.send(&metric{Table: "uap", Tags: tags, Fields: fields})
	u.processRadTable(r, tags, *s.RadioTable, *s.RadioTableStats)
	u.processVAPTable(r, tags, *s.VapTable)
}