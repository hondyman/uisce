import { HiveSqlParserListener } from '../../lib';
import { ColumnNameCreateContext, CreateDatabaseStatementContext, CreateFunctionStatementContext, CreateMaterializedViewStatementContext, CreateTableStatementContext, CreateViewStatementContext, DbSchemaNameContext, DbSchemaNameCreateContext, FromInsertStmtContext, FromSelectStmtContext, FunctionNameCreateContext, InsertStmtContext, SelectStatementContext, StatementContext, TableNameContext, TableNameCreateContext, ViewNameContext, ViewNameCreateContext } from '../../lib/hive/HiveSqlParser';
import { EntityCollector } from '../common/entityCollector';
export declare class HiveEntityCollector extends EntityCollector implements HiveSqlParserListener {
    /** ====== Entity Begin */
    exitTableNameCreate(ctx: TableNameCreateContext): void;
    exitTableName(ctx: TableNameContext): void;
    exitColumnNameCreate(ctx: ColumnNameCreateContext): void;
    exitViewNameCreate(ctx: ViewNameCreateContext): void;
    exitViewName(ctx: ViewNameContext): void;
    exitDbSchemaNameCreate(ctx: DbSchemaNameCreateContext): void;
    exitDbSchemaName(ctx: DbSchemaNameContext): void;
    exitFunctionNameCreate(ctx: FunctionNameCreateContext): void;
    /** ==== Statement begin */
    enterStatement(ctx: StatementContext): void;
    exitStatement(): void;
    enterCreateTableStatement(ctx: CreateTableStatementContext): void;
    exitCreateTableStatement(): void;
    enterSelectStatement(ctx: SelectStatementContext): void;
    exitSelectStatement(ctx: SelectStatementContext): void;
    enterFromSelectStmt(ctx: FromSelectStmtContext): void;
    exitFromSelectStmt(ctx: FromSelectStmtContext): void;
    enterCreateViewStatement(ctx: CreateViewStatementContext): void;
    exitCreateViewStatement(ctx: CreateViewStatementContext): void;
    enterCreateMaterializedViewStatement(ctx: CreateMaterializedViewStatementContext): void;
    exitCreateMaterializedViewStatement(ctx: CreateMaterializedViewStatementContext): void;
    enterInsertStmt(ctx: InsertStmtContext): void;
    exitInsertStmt(ctx: InsertStmtContext): void;
    enterFromInsertStmt(ctx: FromInsertStmtContext): void;
    exitFromInsertStmt(ctx: FromInsertStmtContext): void;
    enterCreateDatabaseStatement(ctx: CreateDatabaseStatementContext): void;
    exitCreateDatabaseStatement(ctx: CreateDatabaseStatementContext): void;
    enterFunctionNameCreate(ctx: FunctionNameCreateContext): void;
    exitCreateFunctionStatement(ctx: CreateFunctionStatementContext): void;
}
