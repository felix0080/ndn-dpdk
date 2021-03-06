#ifndef NDN_DPDK_NDT_NDT_H
#define NDN_DPDK_NDT_NDT_H

/// \file

#include "../ndn/name.h"

/** \brief Per-thread counters for NDT.
 */
typedef struct NdtThread
{
  uint64_t nLookups;
  uint16_t nHits[0];
} NdtThread;

/** \brief The Name Dispatch Table (NDT).
 */
typedef struct Ndt
{
  _Atomic uint8_t* table;
  NdtThread** threads;
  uint64_t indexMask;
  uint64_t sampleMask;
  uint16_t prefixLen;
  uint8_t nThreads;
} Ndt;

/** \brief Initialize NDT.
 *  \param prefixLen prefix length for computing hash.
 *  \param indexBits how many bits to truncate the hash into table entry index.
 *  \param sampleFreq sample once every 2^sampleFreq lookups.
 *  \param nThreads number of lookup threads.
 *  \param sockets array of \p nThreads elements indicating NUMA socket of each lookup thread;
 *                 sockets[0] will be used for the table.
 *  \return array of threads
 */
NdtThread**
Ndt_Init(Ndt* ndt, uint16_t prefixLen, uint8_t indexBits, uint8_t sampleFreq, uint8_t nThreads,
         const unsigned* sockets);

/** \brief Access NdtThread struct.
 */
static inline NdtThread*
Ndt_GetThread(const Ndt* ndt, uint8_t id)
{
  assert(id < ndt->nThreads);
  return ndt->threads[id];
}

/** \brief Update NDT.
 *  \param index table index.
 *  \param value new PIT partition number in the table entry.
 */
void
Ndt_Update(Ndt* ndt, uint64_t index, uint8_t value);

/** \brief Read NDT element.
 */
static inline uint8_t
Ndt_ReadElement(const Ndt* ndt, uint64_t index)
{
  return atomic_load_explicit(&ndt->table[index], memory_order_relaxed);
}

/** \brief Query NDT without counting.
 */
static inline uint8_t
Ndt_Lookup(const Ndt* ndt, const PName* name, const uint8_t* nameV, uint64_t* index)
{
  uint16_t prefixLen = RTE_MIN(name->nComps, ndt->prefixLen);
  uint64_t hash = PName_ComputePrefixHashUncached(name, nameV, prefixLen);
  // Compute hash in 'uncached' mode, to reduce workload of FwInput thread
  *index = hash & ndt->indexMask;
  return Ndt_ReadElement(ndt, *index);
}

static inline uint8_t
Ndtt_Lookup_(const Ndt* ndt, NdtThread* ndtt, const PName* name, const uint8_t* nameV)
{
  uint64_t index;
  uint8_t value = Ndt_Lookup(ndt, name, nameV, &index);
  if ((++ndtt->nLookups & ndt->sampleMask) == 0) {
    ++ndtt->nHits[index];
  }
  return value;
}

/** \brief Query NDT with counting.
 */
static inline uint8_t
Ndtt_Lookup(const Ndt* ndt, NdtThread* ndtt, const Name* name)
{
  return Ndtt_Lookup_(ndt, ndtt, &name->p, name->v);
}

#endif // NDN_DPDK_NDT_NDT_H
