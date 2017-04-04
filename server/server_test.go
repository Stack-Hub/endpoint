/* Copyright 2017, Ashish Thakwani. 
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.LICENSE file.
 */
package server

import (
    "testing"
    "fmt"
    "os"
    "os/exec"
    "strconv"
    "encoding/json"
    "net"
    "runtime"
    "time"
    
    "../utils"
    ps "github.com/shirou/gopsutil/process"
    netutil "github.com/shirou/gopsutil/net"
)

var data = []utils.Host {
            {1001, "192..168.1.1", 2001, 8000, utils.Config{8000}, 1000, 1},
            {1002, "192..168.1.2", 2002, 8000, utils.Config{8000}, 1000, 2},
            {1003, "192..168.1.3", 2003, 8000, utils.Config{8000}, 1000, 3},
            {1004, "192..168.1.4", 2004, 8000, utils.Config{8000}, 1000, 4},
            {1005, "192..168.1.5", 2005, 8000, utils.Config{8000}, 1000, 5},
            {1006, "192..168.1.6", 2006, 8000, utils.Config{8000}, 1000, 6},
            {1007, "192..168.1.7", 2007, 8000, utils.Config{8000}, 1000, 7},
            {1008, "192..168.1.8", 2008, 8000, utils.Config{8000}, 1000, 8},
            {1009, "192..168.1.9", 2009, 8000, utils.Config{8000}, 1000, 9},
            {1010, "192..168.1.10", 2010, 8000, utils.Config{8000}, 1000, 10},
            {1011, "192..168.1.11", 2011, 8000, utils.Config{8000}, 1000, 11},
            {1012, "192..168.1.12", 2012, 8000, utils.Config{8000}, 1000, 12},
            {1013, "192..168.1.13", 2013, 8000, utils.Config{8000}, 1000, 13},
            {1014, "192..168.1.14", 2014, 8000, utils.Config{8000}, 1000, 14},
            {1015, "192..168.1.15", 2015, 8000, utils.Config{8000}, 1000, 15},
            {1016, "192..168.1.16", 2016, 8000, utils.Config{8000}, 1000, 16},
            {1017, "192..168.1.17", 2017, 8000, utils.Config{8000}, 1000, 17},
            {1018, "192..168.1.18", 2018, 8000, utils.Config{8000}, 1000, 18},
            {1019, "192..168.1.19", 2019, 8000, utils.Config{8000}, 1000, 19},
            {1020, "192..168.1.20", 2020, 8000, utils.Config{8000}, 1000, 20},
            {1021, "192..168.1.21", 2021, 8000, utils.Config{8000}, 1000, 21},
            {1022, "192..168.1.22", 2022, 8000, utils.Config{8000}, 1000, 22},
            {1023, "192..168.1.23", 2023, 8000, utils.Config{8000}, 1000, 23},
            {1024, "192..168.1.24", 2024, 8000, utils.Config{8000}, 1000, 24},
            {1025, "192..168.1.25", 2025, 8000, utils.Config{8000}, 1000, 25},
            {1026, "192..168.1.26", 2026, 8000, utils.Config{8000}, 1000, 26},
            {1027, "192..168.1.27", 2027, 8000, utils.Config{8000}, 1000, 27},
            {1028, "192..168.1.28", 2028, 8000, utils.Config{8000}, 1000, 28},
            {1029, "192..168.1.29", 2029, 8000, utils.Config{8000}, 1000, 29},
            {1030, "192..168.1.30", 2030, 8000, utils.Config{8000}, 1000, 30},
            {1031, "192..168.1.31", 2031, 8000, utils.Config{8000}, 1000, 31},
            {1032, "192..168.1.32", 2032, 8000, utils.Config{8000}, 1000, 32},
            {1033, "192..168.1.33", 2033, 8000, utils.Config{8000}, 1000, 33},
            {1034, "192..168.1.34", 2034, 8000, utils.Config{8000}, 1000, 34},
            {1035, "192..168.1.35", 2035, 8000, utils.Config{8000}, 1000, 35},
            {1036, "192..168.1.36", 2036, 8000, utils.Config{8000}, 1000, 36},
            {1037, "192..168.1.37", 2037, 8000, utils.Config{8000}, 1000, 37},
            {1038, "192..168.1.38", 2038, 8000, utils.Config{8000}, 1000, 38},
            {1039, "192..168.1.39", 2039, 8000, utils.Config{8000}, 1000, 39},
            {1040, "192..168.1.40", 2040, 8000, utils.Config{8000}, 1000, 40},
            {1041, "192..168.1.41", 2041, 8000, utils.Config{8000}, 1000, 41},
            {1042, "192..168.1.42", 2042, 8000, utils.Config{8000}, 1000, 42},
            {1043, "192..168.1.43", 2043, 8000, utils.Config{8000}, 1000, 43},
            {1044, "192..168.1.44", 2044, 8000, utils.Config{8000}, 1000, 44},
            {1045, "192..168.1.45", 2045, 8000, utils.Config{8000}, 1000, 45},
            {1046, "192..168.1.46", 2046, 8000, utils.Config{8000}, 1000, 46},
            {1047, "192..168.1.47", 2047, 8000, utils.Config{8000}, 1000, 47},
            {1048, "192..168.1.48", 2048, 8000, utils.Config{8000}, 1000, 48},
            {1049, "192..168.1.49", 2049, 8000, utils.Config{8000}, 1000, 49},
            {1050, "192..168.1.50", 2050, 8000, utils.Config{8000}, 1000, 50},
            {1051, "192..168.1.51", 2051, 8000, utils.Config{8000}, 1000, 51},
            {1052, "192..168.1.52", 2052, 8000, utils.Config{8000}, 1000, 52},
            {1053, "192..168.1.53", 2053, 8000, utils.Config{8000}, 1000, 53},
            {1054, "192..168.1.54", 2054, 8000, utils.Config{8000}, 1000, 54},
            {1055, "192..168.1.55", 2055, 8000, utils.Config{8000}, 1000, 55},
            {1056, "192..168.1.56", 2056, 8000, utils.Config{8000}, 1000, 56},
            {1057, "192..168.1.57", 2057, 8000, utils.Config{8000}, 1000, 57},
            {1058, "192..168.1.58", 2058, 8000, utils.Config{8000}, 1000, 58},
            {1059, "192..168.1.59", 2059, 8000, utils.Config{8000}, 1000, 59},
            {1060, "192..168.1.60", 2060, 8000, utils.Config{8000}, 1000, 60},
            {1061, "192..168.1.61", 2061, 8000, utils.Config{8000}, 1000, 61},
            {1062, "192..168.1.62", 2062, 8000, utils.Config{8000}, 1000, 62},
            {1063, "192..168.1.63", 2063, 8000, utils.Config{8000}, 1000, 63},
            {1064, "192..168.1.64", 2064, 8000, utils.Config{8000}, 1000, 64},
            {1065, "192..168.1.65", 2065, 8000, utils.Config{8000}, 1000, 65},
            {1066, "192..168.1.66", 2066, 8000, utils.Config{8000}, 1000, 66},
            {1067, "192..168.1.67", 2067, 8000, utils.Config{8000}, 1000, 67},
            {1068, "192..168.1.68", 2068, 8000, utils.Config{8000}, 1000, 68},
            {1069, "192..168.1.69", 2069, 8000, utils.Config{8000}, 1000, 69},
            {1070, "192..168.1.70", 2070, 8000, utils.Config{8000}, 1000, 70},
            {1071, "192..168.1.71", 2071, 8000, utils.Config{8000}, 1000, 71},
            {1072, "192..168.1.72", 2072, 8000, utils.Config{8000}, 1000, 72},
            {1073, "192..168.1.73", 2073, 8000, utils.Config{8000}, 1000, 73},
            {1074, "192..168.1.74", 2074, 8000, utils.Config{8000}, 1000, 74},
            {1075, "192..168.1.75", 2075, 8000, utils.Config{8000}, 1000, 75},
            {1076, "192..168.1.76", 2076, 8000, utils.Config{8000}, 1000, 76},
            {1077, "192..168.1.77", 2077, 8000, utils.Config{8000}, 1000, 77},
            {1078, "192..168.1.78", 2078, 8000, utils.Config{8000}, 1000, 78},
            {1079, "192..168.1.79", 2079, 8000, utils.Config{8000}, 1000, 79},
            {1080, "192..168.1.80", 2080, 8000, utils.Config{8000}, 1000, 80},
            {1081, "192..168.1.81", 2081, 8000, utils.Config{8000}, 1000, 81},
            {1082, "192..168.1.82", 2082, 8000, utils.Config{8000}, 1000, 82},
            {1083, "192..168.1.83", 2083, 8000, utils.Config{8000}, 1000, 83},
            {1084, "192..168.1.84", 2084, 8000, utils.Config{8000}, 1000, 84},
            {1085, "192..168.1.85", 2085, 8000, utils.Config{8000}, 1000, 85},
            {1086, "192..168.1.86", 2086, 8000, utils.Config{8000}, 1000, 86},
            {1087, "192..168.1.87", 2087, 8000, utils.Config{8000}, 1000, 87},
            {1088, "192..168.1.88", 2088, 8000, utils.Config{8000}, 1000, 88},
            {1089, "192..168.1.89", 2089, 8000, utils.Config{8000}, 1000, 89},
            {1090, "192..168.1.90", 2090, 8000, utils.Config{8000}, 1000, 90},
            {1091, "192..168.1.91", 2091, 8000, utils.Config{8000}, 1000, 91},
            {1092, "192..168.1.92", 2092, 8000, utils.Config{8000}, 1000, 92},
            {1093, "192..168.1.93", 2093, 8000, utils.Config{8000}, 1000, 93},
            {1094, "192..168.1.94", 2094, 8000, utils.Config{8000}, 1000, 94},
            {1095, "192..168.1.95", 2095, 8000, utils.Config{8000}, 1000, 95},
            {1096, "192..168.1.96", 2096, 8000, utils.Config{8000}, 1000, 96},
            {1097, "192..168.1.97", 2097, 8000, utils.Config{8000}, 1000, 97},
            {1098, "192..168.1.98", 2098, 8000, utils.Config{8000}, 1000, 98},
            {1099, "192..168.1.99", 2099, 8000, utils.Config{8000}, 1000, 99},
            {1100, "192..168.1.100", 2100, 8000, utils.Config{8000}, 1000, 100},
            {1101, "192..168.1.101", 2101, 8000, utils.Config{8000}, 1000, 101},
            {1102, "192..168.1.102", 2102, 8000, utils.Config{8000}, 1000, 102},
            {1103, "192..168.1.103", 2103, 8000, utils.Config{8000}, 1000, 103},
            {1104, "192..168.1.104", 2104, 8000, utils.Config{8000}, 1000, 104},
            {1105, "192..168.1.105", 2105, 8000, utils.Config{8000}, 1000, 105},
            {1106, "192..168.1.106", 2106, 8000, utils.Config{8000}, 1000, 106},
            {1107, "192..168.1.107", 2107, 8000, utils.Config{8000}, 1000, 107},
            {1108, "192..168.1.108", 2108, 8000, utils.Config{8000}, 1000, 108},
            {1109, "192..168.1.109", 2109, 8000, utils.Config{8000}, 1000, 109},
            {1110, "192..168.1.110", 2110, 8000, utils.Config{8000}, 1000, 110},
            {1111, "192..168.1.111", 2111, 8000, utils.Config{8000}, 1000, 111},
            {1112, "192..168.1.112", 2112, 8000, utils.Config{8000}, 1000, 112},
            {1113, "192..168.1.113", 2113, 8000, utils.Config{8000}, 1000, 113},
            {1114, "192..168.1.114", 2114, 8000, utils.Config{8000}, 1000, 114},
            {1115, "192..168.1.115", 2115, 8000, utils.Config{8000}, 1000, 115},
            {1116, "192..168.1.116", 2116, 8000, utils.Config{8000}, 1000, 116},
            {1117, "192..168.1.117", 2117, 8000, utils.Config{8000}, 1000, 117},
            {1118, "192..168.1.118", 2118, 8000, utils.Config{8000}, 1000, 118},
            {1119, "192..168.1.119", 2119, 8000, utils.Config{8000}, 1000, 119},
            {1120, "192..168.1.120", 2120, 8000, utils.Config{8000}, 1000, 120},
            {1121, "192..168.1.121", 2121, 8000, utils.Config{8000}, 1000, 121},
            {1122, "192..168.1.122", 2122, 8000, utils.Config{8000}, 1000, 122},
            {1123, "192..168.1.123", 2123, 8000, utils.Config{8000}, 1000, 123},
            {1124, "192..168.1.124", 2124, 8000, utils.Config{8000}, 1000, 124},
            {1125, "192..168.1.125", 2125, 8000, utils.Config{8000}, 1000, 125},
            {1126, "192..168.1.126", 2126, 8000, utils.Config{8000}, 1000, 126},
            {1127, "192..168.1.127", 2127, 8000, utils.Config{8000}, 1000, 127},
            {1128, "192..168.1.128", 2128, 8000, utils.Config{8000}, 1000, 128},
            {1129, "192..168.1.129", 2129, 8000, utils.Config{8000}, 1000, 129},
            {1130, "192..168.1.130", 2130, 8000, utils.Config{8000}, 1000, 130},
            {1131, "192..168.1.131", 2131, 8000, utils.Config{8000}, 1000, 131},
            {1132, "192..168.1.132", 2132, 8000, utils.Config{8000}, 1000, 132},
            {1133, "192..168.1.133", 2133, 8000, utils.Config{8000}, 1000, 133},
            {1134, "192..168.1.134", 2134, 8000, utils.Config{8000}, 1000, 134},
            {1135, "192..168.1.135", 2135, 8000, utils.Config{8000}, 1000, 135},
            {1136, "192..168.1.136", 2136, 8000, utils.Config{8000}, 1000, 136},
            {1137, "192..168.1.137", 2137, 8000, utils.Config{8000}, 1000, 137},
            {1138, "192..168.1.138", 2138, 8000, utils.Config{8000}, 1000, 138},
            {1139, "192..168.1.139", 2139, 8000, utils.Config{8000}, 1000, 139},
            {1140, "192..168.1.140", 2140, 8000, utils.Config{8000}, 1000, 140},
            {1141, "192..168.1.141", 2141, 8000, utils.Config{8000}, 1000, 141},
            {1142, "192..168.1.142", 2142, 8000, utils.Config{8000}, 1000, 142},
            {1143, "192..168.1.143", 2143, 8000, utils.Config{8000}, 1000, 143},
            {1144, "192..168.1.144", 2144, 8000, utils.Config{8000}, 1000, 144},
            {1145, "192..168.1.145", 2145, 8000, utils.Config{8000}, 1000, 145},
            {1146, "192..168.1.146", 2146, 8000, utils.Config{8000}, 1000, 146},
            {1147, "192..168.1.147", 2147, 8000, utils.Config{8000}, 1000, 147},
            {1148, "192..168.1.148", 2148, 8000, utils.Config{8000}, 1000, 148},
            {1149, "192..168.1.149", 2149, 8000, utils.Config{8000}, 1000, 149},
            {1150, "192..168.1.150", 2150, 8000, utils.Config{8000}, 1000, 150},
            {1151, "192..168.1.151", 2151, 8000, utils.Config{8000}, 1000, 151},
            {1152, "192..168.1.152", 2152, 8000, utils.Config{8000}, 1000, 152},
            {1153, "192..168.1.153", 2153, 8000, utils.Config{8000}, 1000, 153},
            {1154, "192..168.1.154", 2154, 8000, utils.Config{8000}, 1000, 154},
            {1155, "192..168.1.155", 2155, 8000, utils.Config{8000}, 1000, 155},
            {1156, "192..168.1.156", 2156, 8000, utils.Config{8000}, 1000, 156},
            {1157, "192..168.1.157", 2157, 8000, utils.Config{8000}, 1000, 157},
            {1158, "192..168.1.158", 2158, 8000, utils.Config{8000}, 1000, 158},
            {1159, "192..168.1.159", 2159, 8000, utils.Config{8000}, 1000, 159},
            {1160, "192..168.1.160", 2160, 8000, utils.Config{8000}, 1000, 160},
            {1161, "192..168.1.161", 2161, 8000, utils.Config{8000}, 1000, 161},
            {1162, "192..168.1.162", 2162, 8000, utils.Config{8000}, 1000, 162},
            {1163, "192..168.1.163", 2163, 8000, utils.Config{8000}, 1000, 163},
            {1164, "192..168.1.164", 2164, 8000, utils.Config{8000}, 1000, 164},
            {1165, "192..168.1.165", 2165, 8000, utils.Config{8000}, 1000, 165},
            {1166, "192..168.1.166", 2166, 8000, utils.Config{8000}, 1000, 166},
            {1167, "192..168.1.167", 2167, 8000, utils.Config{8000}, 1000, 167},
            {1168, "192..168.1.168", 2168, 8000, utils.Config{8000}, 1000, 168},
            {1169, "192..168.1.169", 2169, 8000, utils.Config{8000}, 1000, 169},
            {1170, "192..168.1.170", 2170, 8000, utils.Config{8000}, 1000, 170},
            {1171, "192..168.1.171", 2171, 8000, utils.Config{8000}, 1000, 171},
            {1172, "192..168.1.172", 2172, 8000, utils.Config{8000}, 1000, 172},
            {1173, "192..168.1.173", 2173, 8000, utils.Config{8000}, 1000, 173},
            {1174, "192..168.1.174", 2174, 8000, utils.Config{8000}, 1000, 174},
            {1175, "192..168.1.175", 2175, 8000, utils.Config{8000}, 1000, 175},
            {1176, "192..168.1.176", 2176, 8000, utils.Config{8000}, 1000, 176},
            {1177, "192..168.1.177", 2177, 8000, utils.Config{8000}, 1000, 177},
            {1178, "192..168.1.178", 2178, 8000, utils.Config{8000}, 1000, 178},
            {1179, "192..168.1.179", 2179, 8000, utils.Config{8000}, 1000, 179},
            {1180, "192..168.1.180", 2180, 8000, utils.Config{8000}, 1000, 180},
            {1181, "192..168.1.181", 2181, 8000, utils.Config{8000}, 1000, 181},
            {1182, "192..168.1.182", 2182, 8000, utils.Config{8000}, 1000, 182},
            {1183, "192..168.1.183", 2183, 8000, utils.Config{8000}, 1000, 183},
            {1184, "192..168.1.184", 2184, 8000, utils.Config{8000}, 1000, 184},
            {1185, "192..168.1.185", 2185, 8000, utils.Config{8000}, 1000, 185},
            {1186, "192..168.1.186", 2186, 8000, utils.Config{8000}, 1000, 186},
            {1187, "192..168.1.187", 2187, 8000, utils.Config{8000}, 1000, 187},
            {1188, "192..168.1.188", 2188, 8000, utils.Config{8000}, 1000, 188},
            {1189, "192..168.1.189", 2189, 8000, utils.Config{8000}, 1000, 189},
            {1190, "192..168.1.190", 2190, 8000, utils.Config{8000}, 1000, 190},
            {1191, "192..168.1.191", 2191, 8000, utils.Config{8000}, 1000, 191},
            {1192, "192..168.1.192", 2192, 8000, utils.Config{8000}, 1000, 192},
            {1193, "192..168.1.193", 2193, 8000, utils.Config{8000}, 1000, 193},
            {1194, "192..168.1.194", 2194, 8000, utils.Config{8000}, 1000, 194},
            {1195, "192..168.1.195", 2195, 8000, utils.Config{8000}, 1000, 195},
            {1196, "192..168.1.196", 2196, 8000, utils.Config{8000}, 1000, 196},
            {1197, "192..168.1.197", 2197, 8000, utils.Config{8000}, 1000, 197},
            {1198, "192..168.1.198", 2198, 8000, utils.Config{8000}, 1000, 198},
            {1199, "192..168.1.199", 2199, 8000, utils.Config{8000}, 1000, 199},
    }


func blockForever() {
    for {
        runtime.Gosched()
    }
}

/*
 * Lock file
 */
func lockFile(pid int) (*exec.Cmd) {
    f := strconv.Itoa(pid)
    cmd := "flock"
    args := []string{utils.RUNPATH + f, "-c", "sleep infinity"}
    
    c := exec.Command(cmd, args...)

    go func () {
        c.Output()
    }()

    return c
}

/*
 * Remove file
 */
func removeFile(f string, t *testing.T) {
    cmd := "rm"
    args := []string{f}
    
    _, err := exec.Command(cmd, args...).Output()
    if err != nil {
        t.Error(
            "\nFailed to remove ", f,
            "\nwith error ", err,
        )        
    }
}

/*
 * Write host data on server socket.
 */
func writeHost(pid int, h *utils.Host) {

    // Form Unix socket based on pid 
    f := utils.RUNPATH + strconv.Itoa(pid) + ".sock"
    c, err := net.Dial("unix", f)
    utils.Check(err)
    
    defer c.Close()

    // Convert host var to json and send to server
    payload, err := json.Marshal(h)
    utils.Check(err)
    
    // Send to server over unix socket.
    _, err = c.Write(payload)
    utils.Check(err)
}


func TestConnections(t *testing.T) {
    
    cmds  := make(map[*exec.Cmd]int)
    hosts := make(map[int]*utils.Host)
    
    pid := os.Getpid()
    
    ConnAddEv := func(p int, h *utils.Host) {
        fmt.Printf("Connected %s:%d on Port %d\n", 
                   h.RemoteIP, 
                   h.AppPort, 
                   h.ListenPort)
        hosts[h.Pid] = h
    }

    ConnRemoveEv := func(p int, h *utils.Host) {
        fmt.Printf("Removed %s:%d from Port %d\n", 
                   h.RemoteIP, 
                   h.AppPort, 
                   h.ListenPort)
        delete(hosts, h.Pid)
        
        f := strconv.Itoa(h.Pid)
        f = utils.RUNPATH + f

        removeFile(f, t)
    }
    
    for _, host := range data {
        cmd := lockFile(host.Pid)
        cmds[cmd] = host.Pid
    }

    
    go func() {
        Monitor(ConnAddEv, ConnRemoveEv)
    }()

    // Wait for server socker to open
    found := false
    for {
        conns, err := netutil.ConnectionsPid("unix", int32(pid))
        utils.Check(err)

        for _, c := range conns {
            // Family = 2 indicates IPv4 socket. Store Listen Port
            // in host structure.
            ip := utils.RUNPATH + strconv.Itoa(pid) + ".sock"
            if c.Family == 1 && c.Laddr.IP == ip {
                found = true

                break
            }            
        }    

        runtime.Gosched()
        if found {
            break
        }
    }
    
    // Connect clients to server
    for _, host := range data {
        writeHost(pid, &host)
        
        //Rate limit connection to avoid resource unavailable failure.
        time.Sleep(time.Millisecond/2)
    }

    // Wait for all connections to be established
    for {
        if len(hosts) == len(data) {
            break
        }
        runtime.Gosched()
    }
    
    // Verify if all clients are connected.
    for _, h := range data {
        fmt.Println("Verifying host ", h)
        if *hosts[h.Pid] != h {
            t.Error(
                "\nFor     ", h,
                "\nexpected", h,
                "\ngot     ", *hosts[h.Pid],
            )
        }
    }
    
    
    // Remove all connections.
    for cmd, _ := range cmds {
        proc, err := ps.NewProcess(int32(cmd.Process.Pid))
        utils.Check(err)
        
        children, err := proc.Children()
        utils.Check(err)
        
        for _, child := range children {
            child.Kill()
        }
        cmd.Process.Kill()
    }

    // Wait for all connections to be removed.
    for {
        if len(hosts) == 0 {
            break
        }
        runtime.Gosched()
    }
    
    //Remove socket file
    f := utils.RUNPATH + strconv.Itoa(pid) + ".sock"
    removeFile(f, t)
}
