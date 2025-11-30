"use client";

import {
  Box,
  Card,
  Typography,
  CardContent,
  Table,
  TableHead,
  TableRow,
  TableCell,
  TableBody,
  TextField,
} from "@mui/material";
import { useState } from "react";

type PingSample = {
  from: string;
  target: string;

  // in the unit of milliseconds
  latencyMs: number;
};

export default function Home() {
  const sources = [
    "lax1",
    "vie1",
    "ams1",
    "dal1",
    "fra1",
    "nyc1",
    "hkg1",
    "sgp1",
    "tyo1",
  ];

  const targets = [
    "google.com",
    "github.com",
    "8.8.8.8",
    "x.com",
    "example.com",
    "1.1.1.1",
    "223.5.5.5",
  ];

  const pingSamples: PingSample[] = [];

  // Generate fake ping samples for all source-target combinations
  targets.forEach((target) => {
    sources.forEach((source) => {
      // Generate realistic latency based on source location
      // Base latencies vary by region and target popularity
      let baseLatency = 0;

      // Different targets have different base latencies
      const targetLatency = {
        "google.com": { min: 10, max: 250 },
        "github.com": { min: 15, max: 280 },
        "8.8.8.8": { min: 5, max: 200 },
        "x.com": { min: 12, max: 270 },
        "example.com": { min: 8, max: 220 },
        "1.1.1.1": { min: 5, max: 180 },
        "223.5.5.5": { min: 20, max: 300 },
      }[target] || { min: 10, max: 250 };

      // Regional adjustments (rough estimates based on geographic distance)
      const regionalMultipliers: { [key: string]: number } = {
        lax1: 1.0, // Los Angeles - good connectivity
        nyc1: 1.1, // New York
        dal1: 0.95, // Dallas - central US
        ams1: 0.9, // Amsterdam - excellent connectivity
        fra1: 0.9, // Frankfurt - excellent connectivity
        vie1: 1.0, // Vienna
        hkg1: 1.2, // Hong Kong - slightly higher
        sgp1: 1.15, // Singapore
        tyo1: 1.1, // Tokyo
      };

      const multiplier = regionalMultipliers[source] || 1.0;
      baseLatency =
        (targetLatency.min +
          Math.random() * (targetLatency.max - targetLatency.min)) *
        multiplier;

      // Add some randomness and round to 1 decimal
      const latencyMs = Math.round(baseLatency * 10) / 10;

      pingSamples.push({
        from: source,
        target: target,
        latencyMs: latencyMs,
      });
    });
  });

  // Helper function to get latency for a specific source-target combination
  const getLatency = (source: string, target: string): number | null => {
    const sample = pingSamples.find(
      (s) => s.from === source && s.target === target
    );
    return sample?.latencyMs ?? null;
  };

  const [target, setTarget] = useState("");

  return (
    <Box sx={{ padding: 2, display: "flex", flexDirection: "column", gap: 2 }}>
      <Card>
        <CardContent>
          <Typography variant="h6">GlobalPing</Typography>
          <Box sx={{ marginTop: 2 }}>
            <TextField
              variant="standard"
              fullWidth
              label="Target"
              value={target}
              onChange={(e) => setTarget(e.target.value)}
            />
          </Box>
        </CardContent>
      </Card>
      <Card>
        <CardContent>
          <Table>
            <TableHead>
              <TableRow>
                <TableCell>Target</TableCell>
                {sources.map((source) => (
                  <TableCell key={source}>{source}</TableCell>
                ))}
              </TableRow>
            </TableHead>
            <TableBody>
              {targets.map((target) => (
                <TableRow key={target}>
                  <TableCell>{target}</TableCell>
                  {sources.map((source) => {
                    const latency = getLatency(source, target);
                    return (
                      <TableCell key={source}>
                        {latency !== null ? `${latency} ms` : "—"}
                      </TableCell>
                    );
                  })}
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </CardContent>
      </Card>
    </Box>
  );
}
