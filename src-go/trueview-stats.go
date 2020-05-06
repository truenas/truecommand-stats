package main

import (
	"fmt"
	"time"
	"encoding/json"
	"os/exec"
	"os"
	"bytes"
	"strings"
	"strconv"
	"context"
)

type GstatSummary struct {
  Name string                               `json:"Name"`
  Lq    float64                             `json:"L(Q)"`
  Ops   float64                             `json:"ops/s"`
  Rs    float64                             `json:"r/s"`
  Rkb   float64                             `json:"kB r"`
  Rkbps float64                             `json:"kBps r"`
  Msr   float64                             `json:"ms/r"`
  Ws    float64                             `json:"w/s"`
  Wkb   float64                             `json:"kB w"`
  Wkbps float64                             `json:"kBps w"`
  Msw   float64                             `json:"ms/w"`
  Busy  float64                             `json:"%busy"`
}

type IfstatSummary struct {
  Name string                               `json:"name"`
  InKB string                               `json:"KB/s in"`
  OutKB string                              `json:"KB/s out"`
}

type ArcstatSummary struct {
  Read float64                              `json:"read"`
  Miss float64                              `json:"miss"`
  MissPerc float64                          `json:"miss%"`
  Dmis float64                              `json:"dmis"`
  DmisPerc float64                          `json:"dm%"`
  Pmis float64                              `json:"pmis"`
  PmisPerc float64                          `json:"pm%"`
  Mmis float64                              `json:"mmis"`
  MmisPerc float64                          `json:"mm%"`
  ArcSz string                              `json:"arcsz"`
  C string                                  `json:"c"`
}

type ServiceSummary struct {
  ClientCount int                           `json:"client_count"`
}

type OutputJson struct {
  Time int64                                `json:"time_t"`
  MemSum interface{}                        `json:"memory_summary,omitempty"`
  VmstatSum interface{}                     `json:"vmstat_summary,omitempty"`
  NetSum interface{}                        `json:"netstat_summary,omitempty"`
  NetUsage []IfstatSummary                  `json:"network_usage,omitempty"`
  ProcStats interface{}                     `json:"process_stats,omitempty"`
  Gstat []GstatSummary                      `json:"gstat_summary,omitempty"`
  ArcStats ArcstatSummary                   `json:"zfs_arcstats,omitempty"`
  TempStats map[string]interface{}          `json:"cpu_temperatures,omitempty"`
  SMB ServiceSummary                        `json:"smb,omitempty"`
  NFS ServiceSummary                        `json:"nfs,omitempty"`
  ISCSI ServiceSummary                      `json:"iscsi,omitempty"`
}

func delete_empty (s []string) []string {
    var r []string
    for _, str := range s {
        if str != "" {
            r = append(r, str)
        }
    }
    return r
}

func ReturnJson( cmd *exec.Cmd , done chan interface{}) {
  //Generic function to output JSON from a command that already returns that format
  var ob bytes.Buffer
  cmd.Stdout = &ob
  err := cmd.Run()
  var ojs interface{}
  if err == nil {
    json.Unmarshal(ob.Bytes(), &ojs);
  }
  done <- ojs
}


func ParseGstat( cmd *exec.Cmd, done chan []GstatSummary) {
  var ob bytes.Buffer
  cmd.Stdout = &ob
  err := cmd.Run()
  var ojs []GstatSummary
  if err != nil { done <- ojs ; return }
  lines := strings.Split(ob.String(), "\n")
  for index, line := range(lines) {
    if(index<2){ continue } //ignore headers
    elem := delete_empty( strings.Split( strings.TrimSpace(line), " ") )
    if len(elem) != 12 { continue; } //not the right number of fields
    var tmp GstatSummary
      tmp.Lq, _ = strconv.ParseFloat(elem[0], 64)
      tmp.Ops, _ = strconv.ParseFloat(elem[1], 64)
      tmp.Rs, _ = strconv.ParseFloat(elem[2], 64)
      tmp.Rkb, _ = strconv.ParseFloat(elem[3], 64)
      tmp.Rkbps, _ = strconv.ParseFloat(elem[4], 64)
      tmp.Msr, _ = strconv.ParseFloat(elem[5], 64)
      tmp.Ws, _ = strconv.ParseFloat(elem[6], 64)
      tmp.Wkb, _ = strconv.ParseFloat(elem[7], 64)
      tmp.Wkbps, _ = strconv.ParseFloat(elem[8], 64)
      tmp.Msw, _ = strconv.ParseFloat(elem[9], 64)
      tmp.Busy, _ = strconv.ParseFloat(elem[10], 64)
      tmp.Name = elem[11]
    ojs = append(ojs, tmp)
  } //end loop over lines
  done <- ojs
}

func ParseArcstat( cmd *exec.Cmd, done chan ArcstatSummary ) {
  // Example FreeNAS Mini output. Note the leading line.

  // dT: 1.005s  w: 1.000s
  // L(q)  ops/s    r/s     kB   kBps   ms/r    w/s     kB   kBps   ms/w   %busy Name
  //    0      0      0      0      0    0.0      0      0      0    0.0    0.0  ada0
  //   72    540      0      0      0    0.0    540    128  69155  135.7  100.0  ada1
  //   52    481      3     12     36   17.2    478    127  60670    9.7   62.3  ada2
  //    0      0      0      0      0    0.0      0      0      0    0.0    0.0  ada3
  //    0      0      0      0      0    0.0      0      0      0    0.0    0.0  ada4
 
  var ob bytes.Buffer
  cmd.Stdout = &ob
  err := cmd.Run()
  var tmp ArcstatSummary
  if err != nil { done <- tmp ; return }
  lines := strings.Split(ob.String(), "\n")
  var headers []string
  for index, line := range(lines) {
    if(index==0) { continue } // Don't need this info
    elem := delete_empty( strings.Split( strings.TrimSpace(line), " ") )
    if(index==1){ headers = elem; continue } //store for a moment
    //Dynamically read the headers to figure out which values go where.
    // Makes it a bit more reliable for changes to the arcstat tool from upstream
    for index, label := range(headers) {
      switch label {
	case "read": tmp.Read, _ = strconv.ParseFloat(elem[index], 64)
	case "miss": tmp.Miss, _ = strconv.ParseFloat(elem[index], 64)
	case "miss%": tmp.MissPerc, _ = strconv.ParseFloat(elem[index], 64)
	case "dmis": tmp.Dmis, _ = strconv.ParseFloat(elem[index], 64)
	case "dm%": tmp.DmisPerc, _ = strconv.ParseFloat(elem[index], 64)
	case "pmis": tmp.Pmis, _ = strconv.ParseFloat(elem[index], 64)
	case "pm%": tmp.PmisPerc, _ = strconv.ParseFloat(elem[index], 64)
	case "mmis": tmp.Mmis, _ = strconv.ParseFloat(elem[index], 64)
	case "mm%": tmp.MmisPerc, _ = strconv.ParseFloat(elem[index], 64)
	case "arcsz": tmp.ArcSz = elem[index]
	case "c": tmp.C = elem[index]
      }
    }
    break
  } //end loop over lines
  done <- tmp
}

func ParseIfstat( cmd *exec.Cmd, done chan []IfstatSummary) {
  var ob bytes.Buffer
  cmd.Stdout = &ob
  err := cmd.Run()
  var ojs []IfstatSummary
  if err != nil { done <- ojs ; return }
  lines := strings.Split(ob.String(), "\n")
  var labels []string
  for index, line := range(lines) {
    if(index==1){ continue } //ignore individual header line
    if index == 0 {
      labels = delete_empty( strings.Split(line, " "))
      continue;
    }
    elem := delete_empty( strings.Split( strings.TrimSpace(line), " ") )
    if len(elem) != (2 * len(labels)) { continue; } //not the right number of fields
    for index, name := range(labels) {
      var tmp IfstatSummary
        tmp.Name = name
        tmp.InKB = elem[ (index*2)]
        tmp.OutKB = elem[ (index*2)+1 ]
      ojs = append(ojs, tmp)
    }
  } //end loop over lines
  done <- ojs
}

func ParseSysctl( cmd *exec.Cmd , filter string, done chan map[string]interface{}) {
  var ob bytes.Buffer
  cmd.Stdout = &ob
  err := cmd.Run()
  ojs := make(map[string]interface{})
  if err != nil { done <- ojs ; return }
  lines := strings.Split(ob.String(), "\n")
  for _, line := range(lines) {
    elem := strings.Split(line, ": ")
    if( len(elem) != 2 ) { continue }
    if(filter != "" && !strings.Contains(elem[0], filter) ){ continue; }
    ctl := strings.Split(elem[0],".")
    ojs[ ctl[ len(ctl)-1] ] = elem[1]
  }
  done <- ojs
}

func ParseSysctlTemperatures( cmd *exec.Cmd , filter string, done chan map[string]interface{}) {
  var ob bytes.Buffer
  cmd.Stdout = &ob
  err := cmd.Run()
  ojs := make(map[string]interface{})
  if err != nil { done <- ojs ; return }
  lines := strings.Split(ob.String(), "\n")
  for _, line := range(lines) {
    elem := strings.Split(line, ": ")
    if( len(elem) != 2 ) { continue }
    if(filter != "" && !strings.Contains(elem[0], filter) ){ continue; }
    ctl := strings.Split(elem[0],".")
    val := strings.TrimSuffix(elem[1], "C")
    ojs[ ctl[ len(ctl)-2] ] = val
  }
  done <- ojs
}

func ParseSMBStatus( cmd *exec.Cmd, done chan ServiceSummary ) {
  // Example output:
  // Samba version 4.12.1
  // PID     Username     Group        Machine                                   Protocol Version  Encryption           Signing              
  // ----------------------------------------------------------------------------------------------------------------------------------------
  // 38807   aervin       aervin       computron9000 (ipv4:192.168.1.232:57030)  NT1               -                    -                    
  // 38806   aervin       aervin       computron9000 (ipv4:192.168.1.232:57002)  NT1  
  // 
  // Chop off the first three lines of output, then count the remaining
  // lines to get the client connection count.
  var ob bytes.Buffer
  cmd.Stdout = &ob
  err := cmd.Run()
  var tmp ServiceSummary
  if err != nil { done <- tmp ; return }
  var lines = strings.Split(ob.String(), "\n")
  startIndex := 0
  tmp.ClientCount = 0
  for index, line := range(lines) {
    if(strings.HasPrefix(line, "--") ){
      startIndex = index //Found the header dividing line
    }else if( startIndex > 0 && line != ""){
      tmp.ClientCount = tmp.ClientCount + 1
    }
  }
  // Other SMB stats can be parsed here in future
  done <- tmp
}

func ParseNFSStatus( cmd *exec.Cmd, done chan ServiceSummary ) {
  // Example output:
  // All mount points on localhost:
  // 10.234.16.36:/mnt/tank_v11/nfs_share
  // 10.234.16.36:/mnt/tank_v11/tc_test_data
  //
  // Note that the same address can have multiple connections
  var ob bytes.Buffer
  cmd.Stdout = &ob
  err := cmd.Run()
  var tmp ServiceSummary
  if err != nil { done <- tmp ; return }
  var lines = strings.Split(ob.String(), "\n")
  if len(lines) < 2 { tmp.ClientCount = 0 ; done <- tmp ; return }
  tmp.ClientCount = 0
  for index,line := range(lines) {
    if( line != "" && index > 0){ tmp.ClientCount = tmp.ClientCount +1 }
  }
  done <- tmp
}

func ParseISCSIStatus( cmd *exec.Cmd, done chan ServiceSummary ) {
  // Example output:
  // ID Portal           Initiator name                       Target name
  // 1 10.234.16.36     iqn.1993-08.org.debian:01:51124594c37 iqn.2005-10.org.freenas.ctl:test.iscsi
  var ob bytes.Buffer
  cmd.Stdout = &ob
  err := cmd.Run()
  var tmp ServiceSummary
  if err != nil { done <- tmp ; return }
  var lines = strings.Split(ob.String(), "\n")
  if len(lines) < 2 { tmp.ClientCount = 0 ; done <- tmp ; return }
  tmp.ClientCount = 0
  for index,line := range(lines) {
    if( line != "" && index >0 ){ tmp.ClientCount = tmp.ClientCount +1 }
  }
  done <- tmp
}

func main() {
  var out OutputJson
  ctx, cancel := context.WithTimeout(context.Background(), 3000*time.Millisecond) //failsafe - kill any process that runs too long
  defer cancel()
  ctxshort, cancelshort := context.WithTimeout(context.Background(), 1100*time.Millisecond) //short failsafe - kill any process that runs too long
  defer cancelshort()

  out.Time = time.Now().Unix()
  //Read in the JSON
  chanA := make(chan interface{})
  go ReturnJson( exec.CommandContext(ctx, "vmstat","-s", "--libxo", "json"), chanA )
  chanB := make(chan interface{})
  go ReturnJson( exec.CommandContext(ctx, "vmstat","-P", "-c", "2", "--libxo", "json"), chanB )
  chanC := make(chan interface{})
  go ReturnJson( exec.CommandContext(ctx, "netstat","-i", "-s", "--libxo", "json"), chanC ) //This always takes 1 second (no adjustments)
  chanD := make(chan []IfstatSummary)
  go ParseIfstat( exec.CommandContext(ctx, "ifstat","-a", "-T", "-b", "1", "1"), chanD ) //Also have this take 1 second (as much data as possible)
  chanE := make(chan interface{})
  go ReturnJson( exec.CommandContext(ctx, "ps","--libxo", "json", "-ax", "-o", "pid,ppid,jail,jid,%cpu,systime,%mem,vsz,rss,state,nlwp,comm"), chanE )
  chanF := make(chan []GstatSummary)
  go ParseGstat( exec.CommandContext(ctx, "gstat", "-bps"), chanF );
  chanG := make(chan ArcstatSummary)
  if _, err := os.Stat("/usr/local/bin/arcstat.py") ; err != nil {
    go ParseArcstat( exec.CommandContext(ctx, "arcstat"), chanG) //FreeNAS 12.0+ and SCALE
  } else {
    go ParseArcstat( exec.CommandContext(ctx, "arcstat.py"), chanG)   //FreeNAS 11.3-
  }
  chanH := make(chan map[string]interface{})
  go ParseSysctlTemperatures( exec.CommandContext(ctx, "sysctl","-q","dev.cpu"), "temperature", chanH)
  chanI := make(chan ServiceSummary)
  go ParseSMBStatus( exec.CommandContext(ctx, "smbstatus","-b"), chanI )
  chanJ := make(chan ServiceSummary)
  go ParseNFSStatus( exec.CommandContext(ctxshort, "showmount","-a", "localhost"), chanJ )
  chanK := make(chan ServiceSummary)
  go ParseISCSIStatus( exec.CommandContext(ctx, "ctladm","islist"), chanK )
  //Assign all the channels to the output fields
  out.MemSum, out.VmstatSum, out.NetSum, out.NetUsage, out.ProcStats, out.Gstat, out.ArcStats, out.TempStats, out.SMB, out.NFS, out.ISCSI = <-chanA, <-chanB, <-chanC, <-chanD, <-chanE, <-chanF, <-chanG, <-chanH, <-chanI, <-chanJ, <-chanK
  //Print out the JSON
  tmp, _ := json.Marshal(out)
  fmt.Println( string(tmp) )
}
