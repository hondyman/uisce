import { ImpalaSqlParserListener } from '../../lib';
import { ColumnNamePathCreateContext, CreateAggregateFunctionContext, CreateFunctionContext, CreateKuduTableAsSelectContext, CreateSchemaContext, CreateTableLikeContext, CreateTableSelectContext, CreateViewContext, DatabaseNameCreateContext, DatabaseNamePathContext, FunctionNameCreateContext, InsertStatementContext, QueryStatementContext, SingleStatementContext, TableNameCreateContext, TableNamePathContext, ViewNameCreateContext, ViewNamePathContext } from '../../lib/impala/ImpalaSqlParser';
import { EntityCollector } from '../common/entityCollector';
export declare class ImpalaEntityCollector extends EntityCollector implements ImpalaSqlParserListener {
    /** ===== Entity begin */
    exitTableNameCreate(ctx: TableNameCreateContext): void;
    exitTableNamePath(ctx: TableNamePathContext): void;
    exitColumnNamePathCreate(ctx: ColumnNamePathCreateContext): void;
    exitViewNameCreate(ctx: ViewNameCreateContext): void;
    exitViewNamePath(ctx: ViewNamePathContext): void;
    exitDatabaseNamePath(ctx: DatabaseNamePathContext): void;
    exitDatabaseNameCreate(ctx: DatabaseNameCreateContext): void;
    exitFunctionNameCreate(ctx: FunctionNameCreateContext): void;
    /** ===== Statement begin */
    enterSingleStatement(ctx: SingleStatementContext): void;
    exitSingleStatement(ctx: SingleStatementContext): void;
    enterCreateTableLike(ctx: CreateTableLikeContext): void;
    exitCreateTableLike(ctx: CreateTableLikeContext): void;
    enterCreateTableSelect(ctx: CreateTableSelectContext): void;
    exitCreateTableSelect(ctx: CreateTableSelectContext): void;
    enterCreateKuduTableAsSelect(ctx: CreateKuduTableAsSelectContext): void;
    exitCreateKuduTableAsSelect(ctx: CreateKuduTableAsSelectContext): void;
    enterQueryStatement(ctx: QueryStatementContext): void;
    exitQueryStatement(ctx: QueryStatementContext): void;
    enterCreateView(ctx: CreateViewContext): void;
    exitCreateView(ctx: CreateViewContext): void;
    enterInsertStatement(ctx: InsertStatementContext): void;
    exitInsertStatement(ctx: InsertStatementContext): void;
    enterCreateSchema(ctx: CreateSchemaContext): void;
    exitCreateSchema(ctx: CreateSchemaContext): void;
    enterCreateAggregateFunction(ctx: CreateAggregateFunctionContext): void;
    exitCreateAggregateFunction(ctx: CreateAggregateFunctionContext): void;
    enterCreateFunction(ctx: CreateFunctionContext): void;
    exitCreateFunction(ctx: CreateFunctionContext): void;
}
