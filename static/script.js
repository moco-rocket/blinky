class BlinkyApp {
    constructor() {
        this.uploadedFiles = [];
        this.initializeElements();
        this.bindEvents();
    }

    initializeElements() {
        this.uploadArea = document.getElementById('uploadArea');
        this.fileInput = document.getElementById('fileInput');
        this.uploadedFilesDiv = document.getElementById('uploadedFiles');
        this.filePreview = document.getElementById('filePreview');
        this.controls = document.getElementById('controls');
        this.processBtn = document.getElementById('processBtn');
        this.resetBtn = document.getElementById('resetBtn');
        this.loading = document.getElementById('loading');
        this.resultArea = document.getElementById('resultArea');
        this.previewImage = document.getElementById('previewImage');
        this.downloadLink = document.getElementById('downloadLink');
        this.messagesDiv = document.getElementById('messages');
        this.formatSelect = document.getElementById('format');
        this.durationInput = document.getElementById('duration');
    }

    bindEvents() {
        // Upload area events
        this.uploadArea.addEventListener('click', () => this.fileInput.click());
        this.uploadArea.addEventListener('dragover', (e) => this.handleDragOver(e));
        this.uploadArea.addEventListener('dragleave', (e) => this.handleDragLeave(e));
        this.uploadArea.addEventListener('drop', (e) => this.handleDrop(e));

        // File input change
        this.fileInput.addEventListener('change', (e) => this.handleFileSelect(e.target.files));

        // Control buttons
        this.processBtn.addEventListener('click', () => this.processImages());
        this.resetBtn.addEventListener('click', () => this.reset());
    }

    handleDragOver(e) {
        e.preventDefault();
        this.uploadArea.classList.add('dragover');
    }

    handleDragLeave(e) {
        e.preventDefault();
        this.uploadArea.classList.remove('dragover');
    }

    handleDrop(e) {
        e.preventDefault();
        this.uploadArea.classList.remove('dragover');
        const files = Array.from(e.dataTransfer.files);
        this.handleFileSelect(files);
    }

    handleFileSelect(files) {
        const validFiles = Array.from(files).filter(file => {
            const validTypes = ['image/jpeg', 'image/jpg', 'image/png', 'image/webp'];
            return validTypes.includes(file.type);
        }).slice(0, 3); // Limit to 3 files

        if (validFiles.length === 0) {
            this.showMessage('有効な画像ファイル（JPG/PNG/WebP）を選択してください。', 'error');
            return;
        }

        this.uploadFiles(validFiles);
    }

    async uploadFiles(files) {
        const formData = new FormData();
        files.forEach(file => {
            formData.append('images', file);
        });

        try {
            this.showLoading(true);
            const response = await fetch('/upload', {
                method: 'POST',
                body: formData
            });

            const result = await response.json();
            
            if (result.success) {
                this.uploadedFiles = result.files;
                this.displayUploadedFiles(files);
                this.showControls();
                this.showMessage('ファイルのアップロードが完了しました！', 'success');
            } else {
                this.showMessage('アップロードに失敗しました。', 'error');
            }
        } catch (error) {
            console.error('Upload error:', error);
            this.showMessage('アップロードエラーが発生しました。', 'error');
        } finally {
            this.showLoading(false);
        }
    }

    displayUploadedFiles(files) {
        this.filePreview.innerHTML = '';
        
        // Ensure we have exactly 3 files for preview (duplicate first if needed)
        const filesToShow = [...files];
        while (filesToShow.length < 3) {
            filesToShow.push(files[0]);
        }

        filesToShow.slice(0, 3).forEach((file, index) => {
            const fileItem = document.createElement('div');
            fileItem.className = 'file-item';

            const img = document.createElement('img');
            img.src = URL.createObjectURL(file);
            img.onload = () => URL.revokeObjectURL(img.src);

            const fileName = document.createElement('div');
            fileName.className = 'file-name';
            fileName.textContent = `フレーム ${index + 1}`;

            fileItem.appendChild(img);
            fileItem.appendChild(fileName);
            this.filePreview.appendChild(fileItem);
        });

        this.uploadedFilesDiv.style.display = 'block';
        this.uploadArea.style.display = 'none';
    }

    showControls() {
        this.controls.style.display = 'block';
    }

    async processImages() {
        const format = this.formatSelect.value;
        const duration = parseFloat(this.durationInput.value);

        const requestData = {
            files: this.uploadedFiles,
            format: format,
            duration: duration
        };

        try {
            this.showLoading(true);
            this.processBtn.disabled = true;

            const response = await fetch('/process', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify(requestData)
            });

            const result = await response.json();

            if (result.success) {
                this.showResult(result.filePath, result.fileName);
                this.showMessage('アニメーションの生成が完了しました！', 'success');
            } else {
                this.showMessage(result.message || 'アニメーション生成に失敗しました。', 'error');
            }
        } catch (error) {
            console.error('Process error:', error);
            this.showMessage('処理エラーが発生しました。', 'error');
        } finally {
            this.showLoading(false);
            this.processBtn.disabled = false;
        }
    }

    showResult(filePath, fileName) {
        this.previewImage.src = filePath;
        this.downloadLink.href = filePath;
        this.downloadLink.download = fileName;
        this.resultArea.style.display = 'block';

        // Scroll to result
        this.resultArea.scrollIntoView({ behavior: 'smooth' });
    }

    showLoading(show) {
        this.loading.style.display = show ? 'block' : 'none';
    }

    showMessage(message, type) {
        const messageDiv = document.createElement('div');
        messageDiv.className = type;
        messageDiv.textContent = message;
        
        this.messagesDiv.innerHTML = '';
        this.messagesDiv.appendChild(messageDiv);

        // Auto-remove success messages
        if (type === 'success') {
            setTimeout(() => {
                if (this.messagesDiv.contains(messageDiv)) {
                    this.messagesDiv.removeChild(messageDiv);
                }
            }, 5000);
        }
    }

    reset() {
        this.uploadedFiles = [];
        this.filePreview.innerHTML = '';
        this.uploadedFilesDiv.style.display = 'none';
        this.uploadArea.style.display = 'block';
        this.controls.style.display = 'none';
        this.resultArea.style.display = 'none';
        this.messagesDiv.innerHTML = '';
        this.fileInput.value = '';
        this.processBtn.disabled = false;
        
        // Reset form values
        this.formatSelect.value = 'apng';
        this.durationInput.value = '0.5';
    }
}

// Initialize the app when DOM is loaded
document.addEventListener('DOMContentLoaded', () => {
    new BlinkyApp();
});