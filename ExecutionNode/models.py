from flask_sqlalchemy import SQLAlchemy

db = SQLAlchemy()

class Project(db.Model):
    """
    项目
    """
    __tablename__ = 'projects'  # 确保表名正确
    id = db.Column(db.Integer, primary_key=True)
    project_hash = db.Column(db.String(50), unique=True, nullable=False)  # 项目唯一标识
    source_node = db.Column(db.Integer, db.ForeignKey('control_nodes.id'), nullable=True) # 来源控制节点
    source_project = db.Column(db.Integer, nullable=True)
    source_workplace = db.Column(db.Integer, nullable=False)
    name = db.Column(db.String(100), nullable=False)
    description = db.Column(db.Text, nullable=True)
    language = db.Column(db.String(50), nullable=False)
    command = db.Column(db.Text, nullable=False)
    mode = db.Column(db.String(10), nullable=False, default='manual')
    created_at = db.Column(db.DateTime, server_default=db.func.now())
    updated_at = db.Column(db.DateTime, server_default=db.func.now(), onupdate=db.func.now())

class Version(db.Model):
    """
    版本
    """
    __tablename__ = 'versions'  # 确保表名正确
    id = db.Column(db.Integer, primary_key=True)
    project_id = db.Column(db.Integer, db.ForeignKey('projects.id'), nullable=False)
    version_hash = db.Column(db.String(100), nullable=False)
    source_node = db.Column(db.Integer, db.ForeignKey('control_nodes.id'), nullable=True)
    description = db.Column(db.Text, nullable=True)
    entrance_file = db.Column(db.String(100), nullable=True)  # 入口文件
    created_at = db.Column(db.DateTime, server_default=db.func.now())
    updated_at = db.Column(db.DateTime, server_default=db.func.now(), onupdate=db.func.now())

class ControlNode(db.Model):
    """
    控制节点
    TODO: 添加worker节点控制节点记录
    """
    __tablename__ = 'control_nodes'  # 确保表名正确
    id = db.Column(db.Integer, primary_key=True)
    name = db.Column(db.String(100), unique=True)
    node_hash = db.Column(db.String(100), unique=True)
    description = db.Column(db.Text, nullable=True)
    status = db.Column(db.String(20), nullable=False, default='inactive')
    ip_address = db.Column(db.String(100), nullable=True)
    rpc_port = db.Column(db.Integer, nullable=False, default=50051)
    auth_key = db.Column(db.String(255), nullable=True)
    created_at = db.Column(db.DateTime, server_default=db.func.now())
    updated_at = db.Column(db.DateTime, server_default=db.func.now(), onupdate=db.func.now())
    is_main = db.Column(db.Boolean, nullable=False, default=False)
    is_enabled = db.Column(db.Boolean, nullable=False, default=True)
    def __str__(self):
        return self.name

class Identity(db.Model):
    """
    当前节点的身份
    """
    __tablename__ = 'identities'  # 确保表名正确
    id = db.Column(db.Integer, primary_key=True)
    identity_hash = db.Column(db.String(100), nullable=False)
    created_at = db.Column(db.DateTime, server_default=db.func.now())
    updated_at = db.Column(db.DateTime, server_default=db.func.now(), onupdate=db.func.now())

class TaskRecord(db.Model):
    """
    记录 Task 的执行情况
    """
    __tablename__ = 'task_records'  # 确保表名正确
    id = db.Column(db.Integer, primary_key=True, autoincrement=True)
    node_hash = db.Column(db.String(20), db.ForeignKey('control_nodes.id'), nullable=False)
    project_hash = db.Column(db.String(20), db.ForeignKey('projects.id'), nullable=False)
    version_hash = db.Column(db.String(20), db.ForeignKey('versions.id'), nullable=False)
    task_id = db.Column(db.Integer, nullable=False)
    task_status = db.Column(db.String(20), nullable=False, default='pending')
    task_chain_id = db.Column(db.Integer, nullable=False)
    task_result = db.Column(db.Text, nullable=True)
    task_created_at = db.Column(db.DateTime, server_default=db.func.now())
    task_updated_at = db.Column(db.DateTime, server_default=db.func.now(), onupdate=db.func.now())
    def __str__(self):
        return f" "