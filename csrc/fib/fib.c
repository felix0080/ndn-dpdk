#include "fib.h"

static int // bool
Fib_LookupMatch(struct cds_lfht_node* lfhtnode, const void* key0)
{
  const FibEntry* entry = container_of(lfhtnode, FibEntry, lfhtnode);
  const LName* key = (const LName*)key0;
  return entry->nameL == key->length && memcmp(entry->nameV, key->value, key->length) == 0;
}

Fib*
Fib_New(const char* id, uint32_t maxEntries, uint32_t nBuckets, unsigned numaSocket,
        uint8_t startDepth)
{
  Fib* fib =
    (Fib*)rte_mempool_create(id, maxEntries, sizeof(FibEntry), 0, sizeof(FibPriv), NULL, NULL, NULL,
                             NULL, numaSocket, MEMPOOL_F_SP_PUT | MEMPOOL_F_SC_GET);
  if (unlikely(fib == NULL)) {
    return NULL;
  }

  FibPriv* fibp = Fib_GetPriv(fib);
  fibp->lfht = cds_lfht_new(nBuckets, nBuckets, nBuckets, 0, NULL);
  if (unlikely(fibp->lfht == NULL)) {
    rte_mempool_free(Fib_ToMempool(fib));
    return NULL;
  }
  fibp->startDepth = startDepth;
  fibp->insertSeqNum = 0;
  return fib;
}

void
Fib_Close(Fib* fib)
{
  FibPriv* fibp = Fib_GetPriv(fib);

  rcu_read_lock();
  struct cds_lfht_iter it;
  struct cds_lfht_node* node;
  cds_lfht_for_each(fibp->lfht, &it, node)
  {
    FibEntry* oldEntry = container_of(node, FibEntry, lfhtnode);
    FibEntry* oldReal = FibEntry_GetReal(oldEntry);
    if (likely(oldReal != NULL)) {
      StrategyCode_Unref(oldReal->strategy);
    }
    cds_lfht_del(fibp->lfht, node);
  }
  rcu_read_unlock();

  int res __rte_unused = cds_lfht_destroy(fibp->lfht, NULL);
  assert(res == 0);
  rte_mempool_free(Fib_ToMempool(fib));
}

bool
Fib_AllocBulk(Fib* fib, FibEntry* entries[], unsigned count)
{
  int res = rte_mempool_get_bulk(Fib_ToMempool(fib), (void**)entries, count);
  if (unlikely(res != 0)) {
    return false;
  }

  for (unsigned i = 0; i < count; ++i) {
    FibEntry* entry = entries[i];
    memset(entry, 0, sizeof(*entry));
    cds_lfht_node_init(&entry->lfhtnode);
  }
  return true;
}

static void
Fib_Free_(FibEntry* entry)
{
  rte_mempool_put(rte_mempool_from_obj(entry), entry);
}

void
Fib_Free(Fib* fib, FibEntry* entry)
{
  Fib_Free_(entry);
}

static void
Fib_RcuFreeVirt(struct rcu_head* rcuhead)
{
  FibEntry* oldVirt = container_of(rcuhead, FibEntry, rcuhead);
  assert(oldVirt->maxDepth > 0);
  assert(oldVirt->nNexthops == 0);
  Fib_Free_(oldVirt);
}

static void
Fib_RcuFreeReal(struct rcu_head* rcuhead)
{
  FibEntry* oldReal = container_of(rcuhead, FibEntry, rcuhead);
  assert(oldReal->maxDepth == 0);
  assert(oldReal->nNexthops > 0);
  Fib_Free_(oldReal);
}

static void
Fib_FreeOld_(FibEntry* entry, Fib_FreeOld freeVirt, Fib_FreeOld freeReal)
{
  FibEntry* oldReal = NULL;
  if (entry != NULL && entry->maxDepth > 0) {
    FibEntry* oldVirt = entry;
    oldReal = oldVirt->realEntry;
    assert(freeVirt != Fib_FreeOld_MustNotExist);
    if (freeVirt == Fib_FreeOld_Yes || freeVirt == Fib_FreeOld_YesIfExists) {
      call_rcu(&oldVirt->rcuhead, Fib_RcuFreeVirt);
    }
  } else {
    oldReal = entry;
    assert(freeVirt == Fib_FreeOld_MustNotExist || freeVirt == Fib_FreeOld_YesIfExists);
  }

  if (oldReal != NULL) {
    assert(freeReal != Fib_FreeOld_MustNotExist);
    // reused entry is not freed but its strategy was ref'ed in Fib_Insert
    StrategyCode_Unref(oldReal->strategy);
    if (freeReal == Fib_FreeOld_Yes || freeReal == Fib_FreeOld_YesIfExists) {
      call_rcu(&oldReal->rcuhead, Fib_RcuFreeReal);
    }
  } else {
    assert(freeReal == Fib_FreeOld_MustNotExist || freeReal == Fib_FreeOld_YesIfExists);
  }
}

void
Fib_Insert(Fib* fib, FibEntry* entry, Fib_FreeOld freeVirt, Fib_FreeOld freeReal)
{
  FibPriv* fibp = Fib_GetPriv(fib);

  FibEntry* newReal = entry;
  if (entry->maxDepth > 0) {
    assert(entry->nNexthops == 0);
    newReal = entry->realEntry;
    entry->seqNum = ++fibp->insertSeqNum;
  }
  if (newReal != NULL) {
    assert(newReal->maxDepth == 0);
    assert(newReal->nNexthops > 0);
    StrategyCode_Ref(newReal->strategy);
    newReal->seqNum = ++fibp->insertSeqNum;
  }

  LName name = { .length = entry->nameL, .value = entry->nameV };
  uint64_t hash = LName_ComputeHash(name);
  struct cds_lfht_node* oldNode =
    cds_lfht_add_replace(fibp->lfht, hash, Fib_LookupMatch, &name, &entry->lfhtnode);
  FibEntry* oldEntry = oldNode == NULL ? NULL : container_of(oldNode, FibEntry, lfhtnode);
  Fib_FreeOld_(oldEntry, freeVirt, freeReal);
}

void
Fib_Erase(Fib* fib, FibEntry* entry, Fib_FreeOld freeVirt, Fib_FreeOld freeReal)
{
  FibPriv* fibp = Fib_GetPriv(fib);
  bool ok = cds_lfht_del(fibp->lfht, &entry->lfhtnode) == 0;
  assert(ok);
  RTE_SET_USED(ok);
  Fib_FreeOld_(entry, freeVirt, freeReal);
}

FibEntry*
Fib_Get(Fib* fib, LName name, uint64_t hash)
{
  FibPriv* fibp = Fib_GetPriv(fib);

  struct cds_lfht_iter it;
  cds_lfht_lookup(fibp->lfht, hash, Fib_LookupMatch, &name, &it);
  struct cds_lfht_node* lfhtnode = cds_lfht_iter_get_node(&it);

  static_assert(offsetof(FibEntry, lfhtnode) == 0,
                ""); // container_of(NULL, FibEntry, lfhtnode) == NULL
  return container_of(lfhtnode, FibEntry, lfhtnode);
}

static FibEntry*
Fib_GetEntryByPrefix(Fib* fib, const PName* name, const uint8_t* nameV, uint16_t prefixLen)
{
  uint64_t hash = PName_ComputePrefixHash(name, nameV, prefixLen);
  return Fib_Get_(fib, PName_SizeofPrefix(name, nameV, prefixLen), nameV, hash);
}

FibEntry*
Fib_Lpm_(Fib* fib, const PName* name, const uint8_t* nameV)
{
  FibPriv* fibp = Fib_GetPriv(fib);

  // first stage
  int prefixLen = name->nComps;
  if (fibp->startDepth < prefixLen) {
    FibEntry* entry = Fib_GetEntryByPrefix(fib, name, nameV, fibp->startDepth);
    if (entry == NULL) { // continue to shorter prefixes
      prefixLen = fibp->startDepth - 1;
    } else if (entry->maxDepth > 0) { // restart at a longest prefix
      prefixLen = fibp->startDepth + entry->maxDepth;
      if (prefixLen > name->nComps) {
        prefixLen = name->nComps;
      }
    } else { // the start entry itself is a match
      return entry;
    }
  }

  // second stage
  for (; prefixLen >= 0; --prefixLen) {
    FibEntry* entry = FibEntry_GetReal(Fib_GetEntryByPrefix(fib, name, nameV, prefixLen));
    if (entry != NULL) {
      return entry;
    }
  }

  return NULL;
}
