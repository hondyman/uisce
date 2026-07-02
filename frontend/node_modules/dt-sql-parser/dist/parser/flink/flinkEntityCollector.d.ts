import { CatalogPathContext, CatalogPathCreateContext, ColumnNameCreateContext, CreateCatalogContext, CreateDatabaseContext, CreateFunctionContext, CreateTableContext, CreateViewContext, DatabasePathContext, DatabasePathCreateContext, FunctionNameCreateContext, InsertStatementContext, QueryStatementContext, SqlStatementContext, TablePathContext, TablePathCreateContext, ViewPathContext, ViewPathCreateContext } from '../../lib/flink/FlinkSqlParser';
import { FlinkSqlParserListener } from '../../lib/flink/FlinkSqlParserListener';
import { EntityCollector } from '../common/entityCollector';
export declare class FlinkEntityCollector extends EntityCollector implements FlinkSqlParserListener {
    /** ====== Entity Begin */
    exitCatalogPathCreate(ctx: CatalogPathCreateContext): void;
    exitCatalogPath(ctx: CatalogPathContext): void;
    exitDatabasePathCreate(ctx: DatabasePathCreateContext): void;
    exitDatabasePath(ctx: DatabasePathContext): void;
    exitTablePath(ctx: TablePathContext): void;
    exitTablePathCreate(ctx: TablePathCreateContext): void;
    exitViewPath(ctx: ViewPathContext): void;
    exitViewPathCreate(ctx: ViewPathCreateContext): void;
    exitColumnNameCreate(ctx: ColumnNameCreateContext): void;
    exitFunctionNameCreate(ctx: FunctionNameCreateContext): void;
    /** ===== Statement begin */
    enterSqlStatement(ctx: SqlStatementContext): void;
    exitSqlStatement(ctx: SqlStatementContext): void;
    enterCreateCatalog(ctx: CreateCatalogContext): void;
    exitCreateCatalog(ctx: CreateCatalogContext): void;
    enterCreateDatabase(ctx: CreateDatabaseContext): void;
    exitCreateDatabase(ctx: CreateDatabaseContext): void;
    enterCreateTable(ctx: CreateTableContext): void;
    exitCreateTable(ctx: CreateTableContext): void;
    enterCreateView(ctx: CreateViewContext): void;
    exitCreateView(ctx: CreateViewContext): void;
    enterQueryStatement(ctx: QueryStatementContext): void;
    exitQueryStatement(ctx: QueryStatementContext): void;
    enterCreateFunction(ctx: CreateFunctionContext): void;
    exitCreateFunction(ctx: CreateFunctionContext): void;
    enterInsertStatement(ctx: InsertStatementContext): void;
    exitInsertStatement(ctx: InsertStatementContext): void;
}
