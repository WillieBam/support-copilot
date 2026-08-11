import { useState, useEffect, useCallback } from 'react';
import type { IncidentTrendPoint, MTTRResult, BreachedIncident } from '@/types/analytics';
import {
  fetchIncidentTrend,
  fetchMTTR,
  fetchBreachedIncidents,
  fetchAllTeamsIncidentTrend,
  fetchAllTeamsMTTR,
  fetchAllTeamsBreachedIncidents,
} from '@/service/analytics/analyticsService';

export type Timeframe = 'day' | 'month' | 'year';
export type SLAFilterOption = 0 | 15 | 30 | 60;

export interface PivotedTrendPoint {
  time_bucket: string;
  OPEN: number;
  IN_PROGRESS: number;
  RESOLVED: number;
  CLOSED: number;
  [key: string]: string | number;
}

// useAnalyticsState manages data fetching, timeframe, sla filtering, and trend data transformation
export function useAnalyticsState(teamId?: string | null, isSuperAdmin?: boolean) {
  const [timeframe, setTimeframe] = useState<Timeframe>('month');
  const [slaTarget, setSlaTarget] = useState<SLAFilterOption>(30);
  const [rawTrend, setRawTrend] = useState<IncidentTrendPoint[]>([]);
  const [pivotedTrend, setPivotedTrend] = useState<PivotedTrendPoint[]>([]);
  const [mttr, setMttr] = useState<MTTRResult | null>(null);
  const [breached, setBreached] = useState<BreachedIncident[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const isAllTeamsMode = Boolean(isSuperAdmin && !teamId);

  const loadData = useCallback(async () => {
    setIsLoading(true);
    setError(null);
    try {
      if (isAllTeamsMode) {
        const [trendData, mttrData, breachedData] = await Promise.all([
          fetchAllTeamsIncidentTrend(timeframe),
          fetchAllTeamsMTTR(slaTarget === 0 ? 30 : slaTarget),
          fetchAllTeamsBreachedIncidents(slaTarget),
        ]);
        setRawTrend(trendData || []);
        setMttr(mttrData || null);
        setBreached(breachedData || []);
      } else if (teamId) {
        const [trendData, mttrData, breachedData] = await Promise.all([
          fetchIncidentTrend(teamId, timeframe),
          fetchMTTR(teamId, slaTarget === 0 ? 30 : slaTarget),
          fetchBreachedIncidents(teamId, slaTarget),
        ]);
        setRawTrend(trendData || []);
        setMttr(mttrData || null);
        setBreached(breachedData || []);
      } else {
        setRawTrend([]);
        setMttr(null);
        setBreached([]);
      }
    } catch (err: any) {
      setError(err?.message || 'Failed to fetch analytics data');
    } finally {
      setIsLoading(false);
    }
  }, [teamId, timeframe, slaTarget, isAllTeamsMode]);

  useEffect(() => {
    loadData();
  }, [loadData]);

  // formatBucketLabel formats raw postgres timestamp string according to selected timeframe
  const formatBucketLabel = (timeBucket: string, tf: Timeframe): string => {
    if (!timeBucket) return '';
    const datePart = timeBucket.split('T')[0].split(' ')[0];
    const parts = datePart.split('-');
    if (parts.length < 3) return datePart;

    const [year, month, day] = parts;
    if (tf === 'year') {
      return year;
    }
    if (tf === 'month') {
      return `${year}-${month}`;
    }
    return `${year}-${month}-${day}`;
  };

  // pivotedTrend transforms flat incident trend rows into stacked recharts series
  useEffect(() => {
    if (!rawTrend.length) {
      setPivotedTrend([]);
      return;
    }

    const bucketsMap: Record<string, PivotedTrendPoint> = {};

    rawTrend.forEach((item) => {
      const label = formatBucketLabel(item.time_bucket, timeframe);
      if (!bucketsMap[label]) {
        bucketsMap[label] = {
          time_bucket: label,
          OPEN: 0,
          IN_PROGRESS: 0,
          RESOLVED: 0,
          CLOSED: 0,
        };
      }
      const statusUpper = item.status ? item.status.toUpperCase() : '';
      if (statusUpper === 'OPEN') {
        bucketsMap[label].OPEN += item.count;
      } else if (statusUpper === 'IN_PROGRESS' || statusUpper === 'IN PRORESS') {
        bucketsMap[label].IN_PROGRESS += item.count;
      } else if (statusUpper === 'RESOLVED') {
        bucketsMap[label].RESOLVED += item.count;
      } else if (statusUpper === 'CLOSED') {
        bucketsMap[label].CLOSED += item.count;
      }
    });

    setPivotedTrend(
      Object.values(bucketsMap).sort((a, b) => a.time_bucket.localeCompare(b.time_bucket))
    );
  }, [rawTrend, timeframe]);

  return {
    teamId,
    isAllTeamsMode,
    timeframe,
    setTimeframe,
    slaTarget,
    setSlaTarget,
    pivotedTrend,
    mttr,
    breached,
    isLoading,
    error,
    refreshAnalytics: loadData,
  };
}
