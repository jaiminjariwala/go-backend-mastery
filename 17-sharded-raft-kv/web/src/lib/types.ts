export interface Entry {
  Delete?: boolean;
  Term: number;
  Key: string;
  Value: string;
  RequestID?: string;
  ReadKey?: string;
}
export interface PeerObservation {
  ID: string;
  RequestID: string;
  Key: string;
  From: number;
  To: number;
  Index: number;
  Term: number;
  Kind: "append" | "ack";
  Delete: boolean;
}
export interface ClientExchange {
  BeforeRows?: Record<number, Record<string, string>>;
  RequestID?: string;
  ID: number;
  Operation: "read" | "write" | "delete";
  Target?: number;
  Key: string;
  Phase: "waiting" | "returned" | "failed";
  ShardID?: number;
  ServerID?: number;
}
export interface Receipt {
  ServerID?: number;
  ShardID?: number;
  Operation: "write" | "read" | "delete";
  RequestID: string;
  Key: string;
  Value: string;
  Index: number;
  Term: number;
  Committed: boolean;
  Replayed: boolean;
  Found: boolean;
}
export interface PendingWrite {
  Operation?: "write" | "delete";
  Key: string;
  Value: string;
  RequestID: string;
}
export interface Server {
  ID: number;
  Role: "follower" | "candidate" | "leader" | "offline";
  Term: number;
  Vote: number;
  Deadline: number;
  Commit: number;
  Up: boolean;
  Log: Entry[] | null;
  KV: Record<string, string>;
  Match: Record<string, number>;
  Snapshot?: {
    Index: number;
    Term: number;
    KV: Record<string, string> | null;
    Requests: Record<
      string,
      { Key: string; Value: string; Index: number; Term: number }
    > | null;
  };
}
export interface Message {
  ID: number;
  From: number;
  To: number;
  Term: number;
  Sent: number;
  Due: number;
  Kind: "vote" | "ballot" | "append" | "snapshot" | "ack";
  Quiet: boolean;
  Log: Entry[] | null;
  Commit: number;
  OK: boolean;
  Index: number;
}
export interface Delivery {
  Sequence: number;
  At: number;
  Status: "delivered" | "dropped";
  Message: Message;
}
export interface WriteProgress {
  Delete?: boolean;
  Key: string;
  Value: string;
  Phase: string;
  Index: number;
  Term: number;
  Confirmed: number[];
  Applied: number[];
  Committed: boolean;
}
export interface ReplicaRuntime {
  PodName: string;
  PodUID: string;
  VolumeClaim: string;
  Boot: string;
  PID: number;
}
export interface Cluster {
  ReplicaRuntime?: ReplicaRuntime[];
  PeerObservations?: PeerObservation[];
  Transport?: string;
  Deployment?: "local" | "kubernetes";
  PeerRPCs?: number[];
  ClientRPCs?: number[];
  ShardID?: number;
  ShardCount?: number;
  Shards?: ShardSummary[];
  Route?: {
    Key: string;
    ShardID: number;
    ShardCount: number;
    Algorithm: string;
  };
  Live?: boolean;
  PIDs?: number[];
  Isolated?: boolean[];
  DroppedRPCs?: number[];
  PartitionTarget?: number | null;
  WALBytes?: number[];
  Durable: boolean;
  Tick: number;
  Serial: number;
  Nodes: Server[];
  Messages: Message[] | null;
  Events: { At: number; Text: string }[];
  Deliveries: Delivery[] | null;
  DeliverySerial: number;
  Progress: WriteProgress | null;
}
export interface ShardSummary {
  ID: number;
  Online: number;
  Connected: number;
  BasePort: number;
  Leader: number | null;
  Term: number;
  Keys: string[];
}
export type Action =
  | { Action: "step" | "reset" | "heal" | "recover" }
  | { Action: "toggle" | "partition" | "snapshot"; ID: number }
  | { Action: "write"; Key: string; Value: string };
