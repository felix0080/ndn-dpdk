export interface Locator {
  Scheme: "udp"|"unixgram"|"tcp"|"unix";
  Local?: string;
  Remote: string;
}
