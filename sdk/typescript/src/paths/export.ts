/* eslint object-curly-newline: ["error", "never"] */
/* eslint max-len: ["error", 160] */
/*
 * This file was generated with makeClass --sdk. Do not edit it.
 */
import * as ApiCallers from '../lib/api_callers';
import { address, Appearance, AppearanceCount, blknum, fourbyte, Function, Log, Monitor, Parameter, Receipt, Statement, Token, topic, Trace, TraceAction, TraceResult, Transaction, uint64 } from '../types';

export function getExport(
  parameters?: {
    addrs: address[],
    topics?: topic[],
    fourbytes?: fourbyte[],
    appearances?: boolean,
    receipts?: boolean,
    logs?: boolean,
    traces?: boolean,
    neighbors?: boolean,
    accounting?: boolean,
    statements?: boolean,
    balances?: boolean,
    withdrawals?: boolean,
    articulate?: boolean,
    cacheTraces?: boolean,
    count?: boolean,
    firstRecord?: uint64,
    maxRecords?: uint64,
    relevant?: boolean,
    emitter?: address[],
    topic?: topic[],
    reverted?: boolean,
    asset?: address[],
    flow?: 'in' | 'out' | 'zero',
    factory?: boolean,
    unripe?: boolean,
    reversed?: boolean,
    noZero?: boolean,
    firstBlock?: blknum,
    lastBlock?: blknum,
    fmt?: string,
    chain: string,
    noHeader?: boolean,
    cache?: boolean,
    decache?: boolean,
    ether?: boolean,
  },
  options?: RequestInit,
) {
  return ApiCallers.fetch<Appearance[] | AppearanceCount[] | Function[] | Log[] | Monitor[] | Parameter[] | Receipt[] | Statement[] | Token[] | Trace[] | TraceAction[] | TraceResult[] | Transaction[]>(
    { endpoint: '/export', method: 'get', parameters, options },
  );
}

