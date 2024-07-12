import { createApi, fetchBaseQuery } from '@reduxjs/toolkit/query/react';
import { createConnectTransport } from "@connectrpc/connect-web";
import { createCallbackClient, createPromiseClient } from '@connectrpc/connect';
import { KeyValueService } from '../grpc/registry/key_value/v1/service_connect';

const key_value_transport = createConnectTransport({
    baseUrl: "http://localhost:2221",
});

const key_value_promise = createPromiseClient(KeyValueService, key_value_transport);

import { GetRequest, GetResponse, GetFilter } from '../grpc/registry/key_value/v1/service_pb';

export const keyValueRPCService = createApi({
    reducerPath: 'key_value_service',
    baseQuery: fetchBaseQuery({ baseUrl: 'http://localhost:2221'}),
    endpoints: (builder) => ({
        getValues: builder.query({
            queryFn: async(req) => { 
                const res = await key_value_promise.get(req)
                return { data: { value: JSON.parse(res.response.value) }}
            }
        }),
        setValue: builder.mutation({
            mutation: async(req) => {
                console.log(req)
            },
        })
    }),
})

export const { useGetValuesQuery } = keyValueRPCService;