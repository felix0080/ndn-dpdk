#ifndef NDN_DPDK_APP_NDNPING_SERVER_H
#define NDN_DPDK_APP_NDNPING_SERVER_H

/// \file

#include "../../core/pcg_basic.h"
#include "../../dpdk/thread.h"
#include "../../iface/face.h"

#define PINGSERVER_MAX_PATTERNS 256
#define PINGSERVER_MAX_REPLIES 8
#define PINGSERVER_MAX_SUM_WEIGHT 256
#define PINGSERVER_BURST_SIZE 64
#define PINGSERVER_PAYLOAD_MAX 65536

typedef uint8_t PingReplyId;

typedef enum PingServerReplyKind
{
  PINGSERVER_REPLY_DATA,
  PINGSERVER_REPLY_NACK,
  PINGSERVER_REPLY_TIMEOUT,
} PingServerReplyKind;

typedef struct PingServerReply
{
  uint64_t nInterests;
  uint32_t freshnessPeriod;
  uint16_t payloadL;
  uint8_t kind;
  uint8_t nackReason;
  LName suffix;
  char suffixBuffer[NAME_MAX_LENGTH];
} PingServerReply;

/** \brief Per-prefix information in ndnping server.
 */
typedef struct PingServerPattern
{
  LName prefix;
  uint16_t nReplies;
  uint16_t nWeights;
  PingReplyId weight[PINGSERVER_MAX_SUM_WEIGHT];
  PingServerReply reply[PINGSERVER_MAX_REPLIES];
  char prefixBuffer[NAME_MAX_LENGTH];
} PingServerPattern;

/** \brief ndnping server.
 */
typedef struct PingServer
{
  struct rte_ring* rxQueue;
  struct rte_mempool* dataMp; ///< mempool for Data
  uint16_t dataMbufHeadroom;
  FaceId face;
  uint16_t nPatterns;
  bool wantNackNoRoute; ///< whether to Nack Interests not matching any pattern

  ThreadStopFlag stop;
  uint64_t nNoMatch;
  uint64_t nAllocError;
  pcg32_random_t replyRng;

  PingServerPattern pattern[PINGSERVER_MAX_PATTERNS];
} PingServer;

void
PingServer_Run(PingServer* server);

#endif // NDN_DPDK_APP_NDNPING_SERVER_H
