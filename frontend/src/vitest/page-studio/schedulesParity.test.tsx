import React from 'react';
import { beforeAll, describe, expect, it, vi } from 'vitest';
import { render } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { MemoryRouter } from 'react-router-dom';
import { loadRuleEngine } from './loadRuleEngine';
import { arrange, scenario } from './schedulesScenario';

/**
 * The Schedules console and the schedule editor, built in Page Studio,
 * against what the hand-built SchedulesPage / ScheduleEditor showed and
 * sent for the same scheduler - recorded below by running the same scenario
 * on them before the console was retired: the list and its search, pause /
 * delete / run now, the run history with its filter and a failed run's
 * reason, and the editor creating a weekly schedule and editing an
 * externally triggered one.
 *
 * Deliberate differences: required selects in the editor show an asterisk;
 * and the weekday picked is the day that runs - the hand-built editor named
 * weekdays in the viewer's time zone from midnight UTC, so west of UTC
 * "Friday" was day 6 (Saturday) in the cron it saved.
 */

const api = vi.hoisted(() => ({ list: vi.fn(), get: vi.fn(), create: vi.fn(), update: vi.fn(), remove: vi.fn(), pause: vi.fn(), resume: vi.fn(), runNow: vi.fn(), kinds: vi.fn(), targets: vi.fn(), calendars: vi.fn(), preview: vi.fn(), runs: vi.fn(), run: vi.fn() }));
vi.mock('../../features/schedules/api', async (orig) => {
  const real = await orig<typeof import('../../features/schedules/api')>();
  return { ...real, schedulesApi: { ...real.schedulesApi, ...api } };
});

import '../../studio-core/registerDomains';
import RuntimePage from '../../pages/page-studio/app/RuntimePage';
import { schedulesBlueprint } from '../../pages/page-studio/app/blueprints/schedules';

beforeAll(loadRuleEngine, 30000);

/** What the hand-built console and editor showed and sent. */
const HAND_BUILT = {
  "list": [
    "DailyNAVreportUpdated<dt>ReportEveryweekdayat18:00(Europe/London)·XLON:skipcloseddays",
    "FactSetloadUpdated<dt>DatapipelineTriggeredbyanexternalscheduler(UTC)·XLON:skipcloseddays",
    "Month-endpackUpdated<dt>SavedqueryBusinessday-1ofeachmonthat06:30(UTC)·XLON:businessdaysonly"
  ],
  "searched": [
    "Month-endpackUpdated<dt>SavedqueryBusinessday-1ofeachmonthat06:30(UTC)·XLON:businessdaysonly"
  ],
  "cleared": 3,
  "paused": "s1",
  "removed": "s2",
  "ranNow": "s3",
  "runs": [
    "<dt>DailyNAVreportSucceeded12rows",
    "<dt>Month-endpackRunnowFailedFailed(SCH-9001)-clickfordetails",
    "<dt>DailyNAVreportSkippedXLONisclosed",
    "<dt>FactSetloadtidal·JOB42Succeeded26read·23written"
  ],
  "downloads": 1,
  "failure": "Thesavedquerynolongerexists.Pickanotherquery.CodeSCH-9001·referencerun2",
  "runFilter": {
    "q": "",
    "status": "failed",
    "kind": "",
    "from": "",
    "to": ""
  },
  "newForm": {
    "labels": [
      "Name",
      "Whattorun",
      "Whichone",
      "Onitsowntimetable",
      "Triggeredbyanexternalscheduler(Tidal,Control-M,AutoSys…)",
      "Repeat",
      "Time",
      "Timezone",
      "Businesscalendar",
      "Runwhatevertheday",
      "Skipdaysthecalendarisclosed",
      "Onaclosedday,runonthenextbusinessday",
      "Active"
    ],
    "inputs": [
      "",
      "",
      "",
      "radio:timetable:true",
      "radio:external:false",
      "weekdays",
      "18:00",
      "America/New_York",
      "",
      "radio:none:true",
      "radio:skip:false",
      "radio:next_business_day:false",
      "checkbox:on:true"
    ]
  },
  "preview": [
    "Runs",
    "Skipped",
    "Moves",
    "Runs",
    "Halfday"
  ],
  "created": {
    "name": "Weekly risk",
    "target": {
      "kind": "report",
      "ref": "r2"
    },
    "timing": {
      "mode": "timetable",
      "cron": "15 7 * * 6",
      "time_zone": "Europe/London",
      "calendar": "XLON",
      "calendar_rule": "next_business_day"
    },
    "enabled": true
  },
  "editForm": {
    "labels": [
      "Name",
      "Whattorun",
      "Whichone",
      "Onitsowntimetable",
      "Triggeredbyanexternalscheduler(Tidal,Control-M,AutoSys…)",
      "Timezone",
      "Businesscalendar",
      "Runwhatevertheday",
      "Skipdaysthecalendarisclosed",
      "Active"
    ],
    "inputs": [
      "FactSetload",
      "data_pipeline",
      "p1",
      "radio:timetable:false",
      "radio:external:true",
      "UTC",
      "XLON",
      "radio:none:false",
      "radio:skip:true",
      "checkbox:on:false"
    ],
    "command": true
  },
  "updated": [
    "s2",
    {
      "name": "FactSet load",
      "target": {
        "kind": "data_pipeline",
        "ref": "p1"
      },
      "timing": {
        "mode": "external",
        "cron": "",
        "time_zone": "UTC",
        "calendar": "XLON",
        "calendar_rule": "skip"
      },
      "enabled": true
    }
  ]
} as const;

const unstar = (labels: readonly string[]) => labels.map((l) => l.replace(/\*$/, ''));

describe('Schedules console built in Page Studio', () => {
  it('lists, filters, pauses, deletes, runs, shows history and edits schedules as the hand-built console did', async () => {
    arrange(api);
    api.get.mockImplementation(async (id: string) => (await api.list()).schedules.find((x: { id: string }) => x.id === id));
    const bp = schedulesBlueprint();
    render(
      <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
        <MemoryRouter>
          <RuntimePage name={bp.name} slug={bp.slug} tabs={bp.tabs!} filterBar={bp.filterBar} components={bp.components} dataSources={[]} tenantId="t1" app={bp.app} />
        </MemoryRouter>
      </QueryClientProvider>,
    );
    const got = await scenario(api) as Record<string, unknown>;
    const want = HAND_BUILT as unknown as Record<string, unknown>;
    for (const k of ['list', 'searched', 'cleared', 'paused', 'removed', 'ranNow', 'runs', 'downloads', 'failure', 'runFilter', 'preview', 'updated']) {
      expect({ [k]: got[k] }).toEqual({ [k]: want[k] });
    }
    const form = (x: unknown) => ({ ...(x as { labels: string[] }), labels: unstar((x as { labels: string[] }).labels) });
    expect(form(got.newForm)).toEqual(form(want.newForm));
    expect(form(got.editForm)).toEqual(form(want.editForm));
    // Friday is day 5 (the recording saved 6 - Saturday - from a mislabelled weekday).
    const created = got.created as { timing: { cron: string } };
    expect(created).toEqual({ ...(want.created as object), timing: { ...(want.created as { timing: object }).timing, cron: '15 7 * * 5' } });
  }, 60000);
});
