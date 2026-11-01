import type { InviteReport, Mismatch, StashProgress } from "../lib/api";
import { formatDateTime } from "../lib/format";

function Section({
  title,
  time,
  children,
}: {
  title: string;
  time?: number;
  children: React.ReactNode;
}) {
  return (
    <section className="max-h-72 overflow-y-auto rounded-box bborder bg-base-200 p-3">
      <div className="mb-2 flex items-baseline justify-between gap-2">
        <h3 className="font-semibold">{title}</h3>
        {time !== undefined && (
          <span className="text-xs opacity-50">{formatDateTime(time)}</span>
        )}
      </div>
      {children}
    </section>
  );
}

export function CharacterReportView({
  mismatches,
  time,
}: {
  mismatches: Mismatch[];
  time: number;
}) {
  return (
    <Section
      title={
        mismatches.length === 1
          ? "1 mismatch"
          : `${mismatches.length} mismatches`
      }
      time={time}
    >
      {mismatches.length === 0 ? (
        <p className="text-sm text-success">No mismatches found.</p>
      ) : (
        <table className="table table-sm">
          <thead>
            <tr>
              <th>Character</th>
              <th>Level</th>
              <th>Team</th>
              <th>Problem</th>
            </tr>
          </thead>
          <tbody>
            {mismatches.map((mismatch) => (
              <tr key={`${mismatch.kind}-${mismatch.character}`}>
                <td>{mismatch.character}</td>
                <td>{mismatch.level}</td>
                <td>{mismatch.team}</td>
                <td>
                  {mismatch.kind === "name"
                    ? "Name does not contain the team abbreviation"
                    : `Invalid ascendancy: ${mismatch.ascendancy}`}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </Section>
  );
}

function NameList({ title, names }: { title: string; names: string[] }) {
  if (names.length === 0) return null;
  return (
    <div>
      <p className="text-sm font-medium">
        {title} ({names.length})
      </p>
      <p className="text-sm break-words opacity-70">{names.join(", ")}</p>
    </div>
  );
}

export function InviteReportView({
  report,
  time,
}: {
  report: InviteReport;
  time: number;
}) {
  return (
    <Section title={report.league} time={time}>
      <div className="flex flex-col gap-2">
        <p className="text-sm">
          {report.requested} requested invites, {report.accepted_members}{" "}
          accepted members
        </p>
        <NameList title="Accepted now" names={report.newly_accepted} />
        <NameList
          title="Requesting but not sorted yet"
          names={report.unsorted_requests}
        />
        <NameList
          title="Accepted but not sorted"
          names={report.unsorted_members}
        />
      </div>
    </Section>
  );
}

export function StashProgressView({ progress }: { progress: StashProgress }) {
  return (
    <Section title="Sync progress">
      <div className="flex flex-col gap-1 text-sm">
        {progress.percent !== null && (
          <progress
            className="progress w-full progress-primary"
            value={progress.percent}
            max={100}
          />
        )}
        <p>
          {progress.percent !== null && `${progress.percent}% - `}
          Current: {formatDateTime(progress.current_time * 1000)}
        </p>
        <p className="opacity-70">
          {progress.uploaded_entries} entries uploaded
        </p>
        <p className="opacity-70">Rate limit: {progress.rate_limit}</p>
      </div>
    </Section>
  );
}
