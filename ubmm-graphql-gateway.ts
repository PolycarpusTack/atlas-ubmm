// api-gateway/src/schema/schema.graphql

type Query {
  # Backlog Item Queries
  backlogItem(id: ID!): BacklogItem
  backlogItems(filter: BacklogItemFilter, pagination: PaginationInput): BacklogItemConnection
  backlogItemChildren(parentId: ID!): [BacklogItem!]!
  
  # Metrics Query
  backlogMetrics: BacklogMetrics!
}

type Mutation {
  # Backlog Item Mutations
  createBacklogItem(input: CreateBacklogItemInput!): BacklogItem!
  updateBacklogItem(id: ID!, input: UpdateBacklogItemInput!): BacklogItem!
  deleteBacklogItem(id: ID!): Boolean!
  reorderBacklogItems(input: [ReorderBacklogItemInput!]!): Boolean!
  setExternalId(id: ID!, system: String!, externalId: String!): Boolean!
}

# Types
type BacklogItem {
  id: ID!
  type: ItemType!
  parentId: ID
  title: String!
  description: String
  storyPoints: Int
  status: ItemStatus!
  priority: Int!
  assignee: String
  tags: [String!]!
  createdAt: DateTime!
  updatedAt: DateTime!
  externalIds: [ExternalId!]!
  
  # Relations
  parent: BacklogItem
  children: [BacklogItem!]!
}

type BacklogItemConnection {
  items: [BacklogItem!]!
  totalCount: Int!
  hasNextPage: Boolean!
  nextPageToken: String
}

type BacklogMetrics {
  totalItems: Int!
  epicCount: Int!
  featureCount: Int!
  storyCount: Int!
  averageAge: Float!
  wipCount: Int!
  leadTimeDays: Float!
  throughputLast30Days: Int!
  icebergRatio: Float!
  healthStatus: HealthStatus!
}

type ExternalId {
  system: String!
  id: String!
}

# Inputs
input BacklogItemFilter {
  types: [ItemType!]
  statuses: [ItemStatus!]
  tags: [String!]
  parentId: ID
  assignee: String
  searchQuery: String
  sortBy: String
  sortOrder: SortOrder
}

input PaginationInput {
  pageSize: Int
  pageToken: String
}

input CreateBacklogItemInput {
  type: ItemType!
  parentId: ID
  title: String!
  description: String
  storyPoints: Int
  status: ItemStatus
  priority: Int
  assignee: String
  tags: [String!]
}

input UpdateBacklogItemInput {
  title: String
  description: String
  type: ItemType
  parentId: ID
  storyPoints: Int
  status: ItemStatus
  priority: Int
  assignee: String
  tags: [String!]
}

input ReorderBacklogItemInput {
  id: ID!
  priority: Int!
}

# Enums
enum ItemType {
  EPIC
  FEATURE
  STORY
}

enum ItemStatus {
  NEW
  READY
  IN_PROGRESS
  BLOCKED
  DONE
}

enum HealthStatus {
  HEALTHY
  AVERAGE
  WARNING
  AT_RISK
}

enum SortOrder {
  ASC
  DESC
}

# Scalars
scalar DateTime

// api-gateway/src/resolvers/backlog.ts

import { BacklogItem, BacklogMetrics, HealthStatus } from '../generated/graphql';
import { backlogServiceClient } from '../clients/backlog-service';

export const backlogResolvers = {
  Query: {
    backlogItem: async (_: any, { id }: { id: string }): Promise<BacklogItem> => {
      return backlogServiceClient.getItem({ id });
    },
    
    backlogItems: async (_: any, { 
      filter, 
      pagination 
    }: { 
      filter?: any, 
      pagination?: { pageSize?: number, pageToken?: string } 
    }) => {
      const response = await backlogServiceClient.listItems({
        types: filter?.types || [],
        statuses: filter?.statuses || [],
        tags: filter?.tags || [],
        parentId: filter?.parentId || '',
        assignee: filter?.assignee || '',
        searchQuery: filter?.searchQuery || '',
        sortBy: filter?.sortBy || '',
        sortOrder: filter?.sortOrder || '',
        pageSize: pagination?.pageSize || 50,
        pageToken: pagination?.pageToken ? parseInt(pagination.pageToken) : 0,
      });
      
      return {
        items: response.items,
        totalCount: response.totalCount,
        hasNextPage: response.nextPageToken > 0,
        nextPageToken: response.nextPageToken > 0 ? response.nextPageToken.toString() : null,
      };
    },
    
    backlogItemChildren: async (_: any, { parentId }: { parentId: string }) => {
      const response = await backlogServiceClient.getChildren({ parentId });
      return response.items;
    },
    
    backlogMetrics: async (): Promise<BacklogMetrics> => {
      const metrics = await backlogServiceClient.getMetrics({});
      return {
        ...metrics,
        healthStatus: metrics.healthStatus as HealthStatus,
      };
    },
  },
  
  Mutation: {
    createBacklogItem: async (_: any, { input }: { input: any }): Promise<BacklogItem> => {
      return backlogServiceClient.createItem({
        type: input.type,
        title: input.title,
        description: input.description || '',
        parentId: input.parentId || '',
        storyPoints: input.storyPoints || 0,
        tags: input.tags || [],
        assignee: input.assignee || '',
      });
    },
    
    updateBacklogItem: async (_: any, { 
      id, 
      input 
    }: { 
      id: string, 
      input: any 
    }): Promise<BacklogItem> => {
      // Transform input for gRPC call
      const updateRequest: any = { id };
      
      if (input.title !== undefined) updateRequest.title = { value: input.title };
      if (input.description !== undefined) updateRequest.description = { value: input.description };
      if (input.status !== undefined) updateRequest.status = { value: input.status };
      if (input.parentId !== undefined) updateRequest.parentId = { value: input.parentId };
      if (input.storyPoints !== undefined) updateRequest.storyPoints = { value: input.storyPoints };
      if (input.priority !== undefined) updateRequest.priority = { value: input.priority };
      if (input.assignee !== undefined) updateRequest.assignee = { value: input.assignee };
      if (input.tags !== undefined) updateRequest.tags = { value: input.tags };
      
      return backlogServiceClient.updateItem(updateRequest);
    },
    
    deleteBacklogItem: async (_: any, { id }: { id: string }): Promise<boolean> => {
      await backlogServiceClient.deleteItem({ id });
      return true;
    },
    
    reorderBacklogItems: async (_: any, { input }: { input: any[] }): Promise<boolean> => {
      await backlogServiceClient.reorderItems({
        items: input.map(item => ({
          id: item.id,
          priority: item.priority,
        })),
      });
      return true;
    },
    
    setExternalId: async (_: any, { 
      id, 
      system, 
      externalId 
    }: { 
      id: string, 
      system: string, 
      externalId: string 
    }): Promise<boolean> => {
      await backlogServiceClient.setExternalId({ id, system, externalId });
      return true;
    },
  },
  
  BacklogItem: {
    parent: async (parent: BacklogItem): Promise<BacklogItem | null> => {
      if (!parent.parentId) return null;
      return backlogServiceClient.getItem({ id: parent.parentId });
    },
    
    children: async (parent: BacklogItem): Promise<BacklogItem[]> => {
      const response = await backlogServiceClient.getChildren({ parentId: parent.id });
      return response.items;
    },
    
    externalIds: (parent: BacklogItem): { system: string, id: string }[] => {
      return Object.entries(parent.externalIds || {}).map(([system, id]) => ({
        system,
        id,
      }));
    },
  },
};

// api-gateway/src/clients/backlog-service.ts

import { credentials } from '@grpc/grpc-js';
import { promisify } from 'util';
import { BacklogServiceClient } from '../generated/proto/backlog_grpc_pb';
import {
  CreateItemRequest,
  GetItemRequest,
  ListItemsRequest,
  UpdateItemRequest,
  DeleteItemRequest,
  GetChildrenRequest,
  ReorderItemsRequest,
  SetExternalIDRequest,
} from '../generated/proto/backlog_pb';
import { Empty } from 'google-protobuf/google/protobuf/empty_pb';

const BACKLOG_SERVICE_URL = process.env.BACKLOG_SERVICE_URL || 'localhost:8080';

// Create gRPC client
const client = new BacklogServiceClient(
  BACKLOG_SERVICE_URL,
  credentials.createInsecure()
);

// Helper to promisify gRPC calls
const promisifyGrpcCall = <TRequest, TResponse>(
  method: (request: TRequest, callback: (error: any, response: TResponse) => void) => void
) => {
  return (request: TRequest): Promise<TResponse> => {
    return new Promise((resolve, reject) => {
      method.call(client, request, (error: any, response: TResponse) => {
        if (error) {
          reject(error);
        } else {
          resolve(response);
        }
      });
    });
  };
};

// Promisify all the client methods
const createItem = promisifyGrpcCall<CreateItemRequest, any>(client.createItem);
const getItem = promisifyGrpcCall<GetItemRequest, any>(client.getItem);
const updateItem = promisifyGrpcCall<UpdateItemRequest, any>(client.updateItem);
const deleteItem = promisifyGrpcCall<DeleteItemRequest, Empty>(client.deleteItem);
const listItems = promisifyGrpcCall<ListItemsRequest, any>(client.listItems);
const getChildren = promisifyGrpcCall<GetChildrenRequest, any>(client.getChildren);
const reorderItems = promisifyGrpcCall<ReorderItemsRequest, Empty>(client.reorderItems);
const setExternalID = promisifyGrpcCall<SetExternalIDRequest, Empty>(client.setExternalID);
const getMetrics = promisifyGrpcCall<Empty, any>(client.getMetrics);

// Export client methods with typed interfaces
export const backlogServiceClient = {
  createItem: async (input: any) => {
    const request = new CreateItemRequest();
    request.setType(input.type);
    request.setTitle(input.title);
    request.setDescription(input.description);
    if (input.parentId) request.setParentId(input.parentId);
    request.setStoryPoints(input.storyPoints);
    if (input.tags && input.tags.length) request.setTagsList(input.tags);
    if (input.assignee) request.setAssignee(input.assignee);
    
    const response = await createItem(request);
    return response.toObject();
  },
  
  getItem: async (input: { id: string }) => {
    const request = new GetItemRequest();
    request.setId(input.id);
    
    const response = await getItem(request);
    return response.toObject();
  },
  
  updateItem: async (input: any) => {
    const request = new UpdateItemRequest();
    request.setId(input.id);
    
    if (input.title) request.setTitle(input.title);
    if (input.description) request.setDescription(input.description);
    if (input.status) request.setStatus(input.status);
    if (input.parentId) request.setParentId(input.parentId);
    if (input.storyPoints) request.setStoryPoints(input.storyPoints);
    if (input.priority) request.setPriority(input.priority);
    if (input.assignee) request.setAssignee(input.assignee);
    if (input.tags) request.setTags(input.tags);
    
    const response = await updateItem(request);
    return response.toObject();
  },
  
  deleteItem: async (input: { id: string }) => {
    const request = new DeleteItemRequest();
    request.setId(input.id);
    
    await deleteItem(request);
    return;
  },
  
  listItems: async (input: any) => {
    const request = new ListItemsRequest();
    
    if (input.types && input.types.length) request.setTypesList(input.types);
    if (input.statuses && input.statuses.length) request.setStatusesList(input.statuses);
    if (input.tags && input.tags.length) request.setTagsList(input.tags);
    if (input.parentId) request.setParentId(input.parentId);
    if (input.assignee) request.setAssignee(input.assignee);
    if (input.searchQuery) request.setSearchQuery(input.searchQuery);
    if (input.pageToken) request.setPageToken(input.pageToken);
    if (input.pageSize) request.setPageSize(input.pageSize);
    if (input.sortBy) request.setSortBy(input.sortBy);
    if (input.sortOrder) request.setSortOrder(input.sortOrder);
    
    const response = await listItems(request);
    return response.toObject();
  },
  
  getChildren: async (input: { parentId: string }) => {
    const request = new GetChildrenRequest();
    request.setParentId(input.parentId);
    
    const response = await getChildren(request);
    return response.toObject();
  },
  
  reorderItems: async (input: { items: { id: string; priority: number }[] }) => {
    const request = new ReorderItemsRequest();
    
    input.items.forEach(item => {
      const reorderItem = new ReorderItemsRequest.ReorderItem();
      reorderItem.setId(item.id);
      reorderItem.setPriority(item.priority);
      request.addItems(reorderItem);
    });
    
    await reorderItems(request);
    return;
  },
  
  setExternalID: async (input: { id: string; system: string; externalId: string }) => {
    const request = new SetExternalIDRequest();
    request.setId(input.id);
    request.setSystem(input.system);
    request.setExternalId(input.externalId);
    
    await setExternalID(request);
    return;
  },
  
  getMetrics: async () => {
    const response = await getMetrics(new Empty());
    return response.toObject();
  },
};

// api-gateway/src/index.ts

import express from 'express';
import { ApolloServer } from '@apollo/server';
import { expressMiddleware } from '@apollo/server/express4';
import { ApolloServerPluginDrainHttpServer } from '@apollo/server/plugin/drainHttpServer';
import http from 'http';
import cors from 'cors';
import bodyParser from 'body-parser';
import { readFileSync } from 'fs';
import { join } from 'path';
import { backlogResolvers } from './resolvers/backlog';

// Load schema
const typeDefs = readFileSync(join(__dirname, 'schema/schema.graphql'), 'utf8');

// Combine resolvers
const resolvers = {
  ...backlogResolvers,
};

async function startApolloServer() {
  // Express app and HTTP server
  const app = express();
  const httpServer = http.createServer(app);
  
  // Create Apollo Server
  const server = new ApolloServer({
    typeDefs,
    resolvers,
    plugins: [ApolloServerPluginDrainHttpServer({ httpServer })],
  });
  
  // Start the server
  await server.start();
  
  // Apply middleware
  app.use(
    '/graphql',
    cors<cors.CorsRequest>(),
    bodyParser.json(),
    expressMiddleware(server, {
      context: async ({ req }) => ({
        token: req.headers.authorization,
      }),
    })
  );
  
  // Add health check endpoint
  app.get('/health', (_, res) => {
    res.status(200).send('OK');
  });
  
  // Start HTTP server
  const PORT = process.env.PORT || 3000;
  httpServer.listen(PORT, () => {
    console.log(`🚀 API Gateway ready at http://localhost:${PORT}/graphql`);
  });
}

// Start the server
startApolloServer().catch((err) => {
  console.error('Failed to start server:', err);
  process.exit(1);
});

// api-gateway/package.json

{
  "name": "ubmm-api-gateway",
  "version": "1.0.0",
  "description": "GraphQL API Gateway for UBMM",
  "main": "dist/index.js",
  "scripts": {
    "start": "node dist/index.js",
    "dev": "nodemon --exec ts-node src/index.ts",
    "build": "tsc",
    "generate": "graphql-codegen --config codegen.yml",
    "lint": "eslint . --ext .ts",
    "test": "jest"
  },
  "dependencies": {
    "@apollo/server": "^4.9.5",
    "@grpc/grpc-js": "^1.9.13",
    "@grpc/proto-loader": "^0.7.10",
    "body-parser": "^1.20.2",
    "cors": "^2.8.5",
    "dotenv": "^16.3.1",
    "express": "^4.18.2",
    "google-protobuf": "^3.21.2",
    "graphql": "^16.8.1",
    "graphql-tag": "^2.12.6",
    "winston": "^3.11.0"
  },
  "devDependencies": {
    "@graphql-codegen/cli": "^5.0.0",
    "@graphql-codegen/typescript": "^4.0.1",
    "@graphql-codegen/typescript-resolvers": "^4.0.1",
    "@types/cors": "^2.8.17",
    "@types/express": "^4.17.21",
    "@types/node": "^20.10.4",
    "@typescript-eslint/eslint-plugin": "^6.13.2",
    "@typescript-eslint/parser": "^6.13.2",
    "eslint": "^8.55.0",
    "jest": "^29.7.0",
    "nodemon": "^3.0.2",
    "ts-jest": "^29.1.1",
    "ts-node": "^10.9.1",
    "typescript": "^5.3.3"
  }
}

// api-gateway/codegen.yml

overwrite: true
schema: "./src/schema/schema.graphql"
generates:
  src/generated/graphql.ts:
    plugins:
      - "typescript"
      - "typescript-resolvers"
    config:
      useIndexSignature: true
      contextType: "../types#Context"
      mappers:
        BacklogItem: "../types#BacklogItemModel"
        BacklogMetrics: "../types#BacklogMetricsModel"

// api-gateway/src/types.ts

export interface Context {
  token?: string;
}

export interface BacklogItemModel {
  id: string;
  type: string;
  parentId?: string;
  title: string;
  description?: string;
  storyPoints?: number;
  status: string;
  priority: number;
  assignee?: string;
  tags: string[];
  createdAt: string;
  updatedAt: string;
  externalIds: Record<string, string>;
}

export interface BacklogMetricsModel {
  totalItems: number;
  epicCount: number;
  featureCount: number;
  storyCount: number;
  averageAge: number;
  wipCount: number;
  leadTimeDays: number;
  throughputLast30Days: number;
  icebergRatio: number;
  healthStatus: string;
}
