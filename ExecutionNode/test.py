import requests

BASE_URL = "http://127.0.0.1:5000/projects"

# 创建项目测试
def test_create_project():
    data = {
        "workplace_id": 1,
        "name": "Sample Project",
        "description": "A sample project description",
        "language": "Python",
        "command": "python app.py",
        "mode": "manual"
    }
    response = requests.post(BASE_URL, json=data)
    print(response.json())

# 获取项目测试
def test_get_project(project_id):
    response = requests.get(f"{BASE_URL}/{project_id}")
    print(response.json())

if __name__ == "__main__":
    test_create_project()
    test_get_project(1)  # 替换为实际 ID
