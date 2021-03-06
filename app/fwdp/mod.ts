import { Counter } from "../../core/mod";
import * as runningStat from "../../core/runningstat/mod";
import * as iface from "../../iface/mod";

export interface InputInfo {
  LCore: number;
  Faces: iface.FaceId[];
}

export interface FwdInfo {
  LCore: number;

  InputInterest: FwdInputCounter;
  InputData: FwdInputCounter;
  InputNack: FwdInputCounter;
  InputLatency: runningStat.Snapshot;

  NNoFibMatch: Counter;
  NDupNonce: Counter;
  NSgNoFwd: Counter;
  NNackMismatch: Counter;

  HeaderMpUsage: Counter;
  IndirectMpUsage: Counter;
}

export interface FwdInputCounter {
  NDropped: Counter;
  NQueued: Counter;
  NCongMarks: Counter;
}
