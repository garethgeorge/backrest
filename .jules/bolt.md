## 2024-05-15 - [Refactored summaryDashboard backup chart loop]
**Learning:** React component with high frequency dashboard needs memoization for derived charting state. Specifically the \`recentBackupsChart\` calculation re-ran on every single render.
**Action:** Used \`useMemo\` to prevent recalculation of the backups chart array on every render, this stops recalculating colors, formatting timestamps, and building the chart data repeatedly unless the \`recentBackups\` data changes.
