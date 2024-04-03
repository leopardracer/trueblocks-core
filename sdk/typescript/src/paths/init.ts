/* eslint object-curly-newline: ["error", "never"] */
/* eslint max-len: ["error", 160] */
// Copyright 2016, 2024 The TrueBlocks Authors. All rights reserved.
// Use of this source code is governed by a license that can
// be found in the LICENSE file.
/*
 * This file was auto generated. DO NOT EDIT.
 */
import * as ApiCallers from '../lib/api_callers';
import { blknum, ChunkRecord, double, Manifest } from '../types';

export function getInit(
  parameters?: {
    all?: boolean,
    dryRun?: boolean,
    firstBlock?: blknum,
    sleep?: double,
    chain: string,
  },
  options?: RequestInit,
) {
  return ApiCallers.fetch<ChunkRecord[] | Manifest[]>(
    { endpoint: '/init', method: 'get', parameters, options },
  );
}
