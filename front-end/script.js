const mat = [
    [0, 0, 0, 0],
    [0, 0, 0, 0],
    [0, 0, 0, 0],
    [0, 0, 0, 0]
];

let firstTime = true;

function canMove(mat, direction) {
    if (direction === 'w') {
        for (let col = 0; col < mat[0].length; col++) {
            for (let row = 1; row < mat.length; row++) {
                if (mat[row][col] !== 0 && (mat[row - 1][col] === 0 || mat[row - 1][col] === mat[row][col])) {
                    return true;
                }
            }
        }
    } else if (direction === 's') {
        for (let col = 0; col < mat[0].length; col++) {
            for (let row = mat.length - 2; row >= 0; row--) {
                if (mat[row][col] !== 0 && (mat[row + 1][col] === 0 || mat[row + 1][col] === mat[row][col])) {
                    return true;
                }
            }
        }
    } else if (direction === 'a') {
        for (let row = 0; row < mat.length; row++) {
            for (let col = 1; col < mat[0].length; col++) {
                if (mat[row][col] !== 0 && (mat[row][col - 1] === 0 || mat[row][col - 1] === mat[row][col])) {
                    return true;
                }
            }
        }
    } else if (direction === 'd') {
        for (let row = 0; row < mat.length; row++) {
            for (let col = mat[0].length - 2; col >= 0; col--) {
                if (mat[row][col] !== 0 && (mat[row][col + 1] === 0 || mat[row][col + 1] === mat[row][col])) {
                    return true;
                }
            }
        }
    }
    return false;
}

function addRandomNumber() {
    const blankSpaces = [];
    for (let row = 0; row < mat.length; row++) {
        for (let col = 0; col < mat[row].length; col++) {
            if (mat[row][col] === 0) {
                blankSpaces.push([row, col]);
            }
        }
    }
    if (blankSpaces.length > 0) {
        const [row, col] = blankSpaces[Math.floor(Math.random() * blankSpaces.length)];
        mat[row][col] = Math.random() < 0.9 ? 2 : 4;
    }
}

function render() {
    const gridContainer = document.getElementById('grid-container');
    gridContainer.innerHTML = ''; // Clear previous cells
    for (let row = 0; row < mat.length; row++) {
        for (let col = 0; col < mat[row].length; col++) {
            const cell = document.createElement('div');
            cell.id = `cell-${row * 4 + col}`;
            cell.className = `grid-cell ${mat[row][col] ? 'tile-' + mat[row][col] : ''}`;
            cell.textContent = mat[row][col] === 0 ? '' : mat[row][col];
            gridContainer.appendChild(cell);
        }
    }
}

function move(direction) {
    if (direction === 'w') {
        for (let col = 0; col < mat[0].length; col++) {
            let merged = Array(mat.length).fill(false);
            for (let row = 1; row < mat.length; row++) {
                if (mat[row][col] !== 0) {
                    let currentRow = row;
                    while (currentRow > 0 && mat[currentRow - 1][col] === 0) {
                        mat[currentRow - 1][col] = mat[currentRow][col];
                        mat[currentRow][col] = 0;
                        currentRow--;
                    }
                    if (currentRow > 0 && mat[currentRow - 1][col] === mat[currentRow][col] && !merged[currentRow - 1]) {
                        mat[currentRow - 1][col] *= 2;
                        mat[currentRow][col] = 0;
                        merged[currentRow - 1] = true;
                    }
                }
            }
        }
    } else if (direction === 's') {
        for (let col = 0; col < mat[0].length; col++) {
            let merged = Array(mat.length).fill(false);
            for (let row = mat.length - 2; row >= 0; row--) {
                if (mat[row][col] !== 0) {
                    let currentRow = row;
                    while (currentRow < mat.length - 1 && mat[currentRow + 1][col] === 0) {
                        mat[currentRow + 1][col] = mat[currentRow][col];
                        mat[currentRow][col] = 0;
                        currentRow++;
                    }
                    if (currentRow < mat.length - 1 && mat[currentRow + 1][col] === mat[currentRow][col] && !merged[currentRow + 1]) {
                        mat[currentRow + 1][col] *= 2;
                        mat[currentRow][col] = 0;
                        merged[currentRow + 1] = true;
                    }
                }
            }
        }
    } else if (direction === 'a') {
        for (let row = 0; row < mat.length; row++) {
            let merged = Array(mat[0].length).fill(false);
            for (let col = 1; col < mat[0].length; col++) {
                if (mat[row][col] !== 0) {
                    let currentCol = col;
                    while (currentCol > 0 && mat[row][currentCol - 1] === 0) {
                        mat[row][currentCol - 1] = mat[row][currentCol];
                        mat[row][currentCol] = 0;
                        currentCol--;
                    }
                    if (currentCol > 0 && mat[row][currentCol - 1] === mat[row][currentCol] && !merged[currentCol - 1]) {
                        mat[row][currentCol - 1] *= 2;
                        mat[row][currentCol] = 0;
                        merged[currentCol - 1] = true;
                    }
                }
            }
        }
    } else if (direction === 'd') {
        for (let row = 0; row < mat.length; row++) {
            let merged = Array(mat[0].length).fill(false);
            for (let col = mat[0].length - 2; col >= 0; col--) {
                if (mat[row][col] !== 0) {
                    let currentCol = col;
                    while (currentCol < mat[0].length - 1 && mat[row][currentCol + 1] === 0) {
                        mat[row][currentCol + 1] = mat[row][currentCol];
                        mat[row][currentCol] = 0;
                        currentCol++;
                    }
                    if (currentCol < mat[0].length - 1 && mat[row][currentCol + 1] === mat[row][currentCol] && !merged[currentCol + 1]) {
                        mat[row][currentCol + 1] *= 2;
                        mat[row][currentCol] = 0;
                        merged[currentCol + 1] = true;
                    }
                }
            }
        }
    }
}

function checkGameOver() {
    if (!canMove(mat, 'w') && !canMove(mat, 's') && !canMove(mat, 'a') && !canMove(mat, 'd')) {
        if (confirm('游戏结束，重新开始新游戏？')) {
            resetGame();
        }
    }
}

function checkWin() {
    for (let row = 0; row < mat.length; row++) {
        for (let col = 0; col < mat[row].length; col++) {
            if (mat[row][col] === 2048) {
                if (confirm('你赢了！冲洗开始新游戏？')) {
                    resetGame();
                }
                return;
            }
        }
    }
}

function resetGame() {
    for (let row = 0; row < mat.length; row++) {
        for (let col = 0; col < mat[row].length; col++) {
            mat[row][col] = 0;
        }
    }
    firstTime = true;
    initializeGame();
}

function initializeGame() {
    if (firstTime) {
        const positions = [];
        for (let row = 0; row < mat.length; row++) {
            for (let col = 0; col < mat[row].length; col++) {
                positions.push([row, col]);
            }
        }
        const randomPositions = positions.sort(() => 0.5 - Math.random()).slice(0, 2);
        for (const [row, col] of randomPositions) {
            mat[row][col] = Math.random() < 0.9 ? 2 : 4;
        }
        firstTime = false;
    }
    render();
}

document.addEventListener('keydown', (event) => {
    const keyMap = {
        'ArrowUp': 'w',
        'ArrowDown': 's',
        'ArrowLeft': 'a',
        'ArrowRight': 'd'
    };
    const direction = keyMap[event.key];
    if (direction && canMove(mat, direction)) {
        move(direction);
        addRandomNumber();
        render();
        checkWin();
        checkGameOver();
    }
});

window.onload = () => {
    initializeGame();
};
