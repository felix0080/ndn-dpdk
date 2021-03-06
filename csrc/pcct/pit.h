#ifndef NDN_DPDK_PCCT_PIT_H
#define NDN_DPDK_PCCT_PIT_H

/// \file

#include "pcct.h"
#include "pit-result.h"

/** \brief Maximum PIT entry lifetime (millis).
 */
#define PIT_MAX_LIFETIME 120000

/** \brief Cast Pcct* as Pit*.
 */
static inline Pit*
Pit_FromPcct(const Pcct* pcct)
{
  return (Pit*)pcct;
}

/** \brief Cast Pit* as Pcct*.
 */
static inline Pcct*
Pit_ToPcct(const Pit* pit)
{
  return (Pcct*)pit;
}

/** \brief Access PitPriv* struct.
 */
static inline PitPriv*
Pit_GetPriv(const Pit* pit)
{
  return &Pcct_GetPriv(Pit_ToPcct(pit))->pitPriv;
}

/** \brief Constructor.
 */
void
Pit_Init(Pit* pit);

/** \brief Get number of PIT entries.
 */
static inline uint32_t
Pit_CountEntries(const Pit* pit)
{
  return Pit_GetPriv(pit)->nEntries;
}

/** \brief Trigger expired timers.
 */
static inline void
Pit_TriggerTimers(Pit* pit)
{
  PitPriv* pitp = Pit_GetPriv(pit);
  MinSched_Trigger(pitp->timeoutSched);
}

/** \brief Set callback when strategy timer expires.
 */
void
Pit_SetSgTimerCb(Pit* pit, Pit_SgTimerCb cb, void* arg);

static inline void
Pit_InvokeSgTimerCb_(Pit* pit, PitEntry* entry)
{
  PitPriv* pitp = Pit_GetPriv(pit);
  (*pitp->sgTimerCb)(pit, entry, pitp->sgTimerCbArg);
}

/** \brief Insert or find a PIT entry for the given Interest.
 *  \param npkt Interest packet.
 *
 *  The PIT-CS lookup includes forwarding hint. PInterest's \c activeFh field
 *  indicates which fwhint is in use, and setting it to -1 disables fwhint.
 *
 *  If there is a CS match, return the CS entry. If there is a PIT match,
 *  return the PIT entry. Otherwise, unless the PCCT is full, insert and
 *  initialize a PIT entry.
 *
 *  When a new PIT entry is inserted, the PIT entry owns \p npkt but does not
 *  free it, so the caller may continue using it until \c PitEntry_InsertDn.
 */
PitInsertResult
Pit_Insert(Pit* pit, Packet* npkt, const FibEntry* fibEntry);

/** \brief Erase a PIT entry.
 *  \post \p entry is no longer valid.
 */
void
Pit_Erase(Pit* pit, PitEntry* entry);

/** \brief Erase both PIT entries on a PccEntry but retain the PccEntry.
 */
void
Pit_RawErase01_(Pit* pit, PccEntry* pccEntry);

/** \brief Find PIT entries matching a Data.
 *  \param npkt Data packet, its token will be used.
 */
PitFindResult
Pit_FindByData(Pit* pit, Packet* npkt);

/** \brief Find PIT entry matching a Nack.
 *  \param npkt Nack packet, its token will be used.
 */
PitEntry*
Pit_FindByNack(Pit* pit, Packet* npkt);

static inline uint64_t
PitEntry_GetToken(PitEntry* entry)
{
  // Declaration is in pit-entry.h.
  PccEntry* pccEntry = PccEntry_FromPitEntry(entry);
  assert(pccEntry->hasToken);
  return pccEntry->token;
}

#endif // NDN_DPDK_PCCT_PIT_H
