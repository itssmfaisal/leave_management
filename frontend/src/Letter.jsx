import { formatDate, formatDays } from "./dates.js";

const dots = (n) => ".".repeat(n);

// The printable A4 leave application. Layout mirrors the paper form.
export default function Letter({ leave, balance }) {
  const holiday = leave.leaveType === "holiday";

  return (
    <div className="letter">
      <p>Date: {formatDate(leave.appliedOn)}</p>
      <p>To</p>
      <p>HRM</p>
      <p>Dhaka Popular Travel Ltd</p>
      <p>Gulshan Badda Link Rd, Dhaka 1212</p>
      <div className="gap" />
      <p>Re: Request for leave</p>
      <div className="gap" />
      <p>Dear Sir,</p>
      <p>I, {dots(20)}{leave.name}{dots(36)}</p>
      <p>Employee Number{dots(12)}{leave.empNo}{dots(30)}</p>
      <div className="gap" />
      <p>
        want holiday from my yearly <span className={`check ${holiday ? "on" : ""}`} /> Holiday or{" "}
        <span className={`check ${holiday ? "" : "on"}`} />Sick Leave.
      </p>

      <table className="details">
        <tbody>
          <tr><td>Last working day</td><td>{formatDate(leave.lastWorkingDay)}</td></tr>
          <tr><td>Holiday starts from</td><td>{formatDate(leave.startDate)}</td></tr>
          <tr><td>Holiday end</td><td>{formatDate(leave.endDate)}</td></tr>
          <tr><td>Number of days</td><td>{formatDays(leave.numDays)}</td></tr>
          <tr><td>Re-joining date</td><td>{formatDate(leave.rejoinDate)}</td></tr>
          <tr className="reason"><td>Reason of leave</td><td>{leave.reason}</td></tr>
        </tbody>
      </table>

      <div className="gap" />
      <p>Please accept my leave application and give me an acknowledgment.</p>
      <div className="gap" />
      <p>Kind Regards</p>
      <div className="gap" />
      <p>Signature:</p>
      <p>Name: {dots(9)}{leave.name}{dots(24)}</p>
      <div className="gap" />
      <div className="gap" />

      <table className="balance">
        <tbody>
          <tr><td>Type</td><td>Eligibility</td><td>Used</td><td>This time</td><td>Remaining</td></tr>
          <tr>
            <td>Casual Leave</td>
            <td className="num">{balance.eligibility}</td>
            <td className="num">{formatDays(balance.used)}</td>
            <td className="num">{formatDays(balance.thisTime)}</td>
            <td className="num">{formatDays(balance.remaining)}</td>
          </tr>
        </tbody>
      </table>
    </div>
  );
}
